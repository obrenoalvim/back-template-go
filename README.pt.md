[English](README.md) | Português

# back-template-go

[![CI](https://github.com/obrenoalvim/back-template-go/actions/workflows/ci.yml/badge.svg)](https://github.com/obrenoalvim/back-template-go/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Template inicial de backend em Go: Fiber, GORM + Postgres + golang-migrate, auth JWT com refresh token rotativo/revogável, rate limiting, logging estruturado e Docker, tudo integrado e testado de ponta a ponta. Faz parte de uma família de templates de backend (`back-template-nest`, `back-template-laravel`, `back-template-spring`, `back-template-fastapi`) que compartilha o mesmo contrato de endpoints e formato de erro entre stacks diferentes.

## Conteúdo

- [Stack](#stack)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Começando (Docker)](#começando-docker--recomendado)
- [Começando (sem Docker)](#começando-sem-docker)
- [Variáveis de ambiente](#variáveis-de-ambiente)
- [Auth](#auth)
- [Papéis (roles)](#papéis-roles)
- [Formato de erro](#formato-de-erro)
- [Banco de dados](#banco-de-dados)
- [Exemplo de recurso CRUD](#exemplo-de-recurso-crud)
- [Testes](#testes)
- [CI/CD](#cicd)
- [Docker](#docker-1)
- [Scripts](#scripts)
- [Usando como template](#usando-como-template)
- [Notas de design e pegadinhas](#notas-de-design-e-pegadinhas)

## Stack

- [Go](https://go.dev) 1.25
- [Fiber](https://gofiber.io) v2: router/middleware estilo Express, o framework web Go mais usado
- [GORM](https://gorm.io) + [pgx](https://github.com/jackc/pgx) (via `gorm.io/driver/postgres`) + [golang-migrate](https://github.com/golang-migrate/migrate): schema versionado em arquivos SQL, embutido no binário via `go:embed`, aplicado automaticamente ao subir
- [Postgres](https://www.postgresql.org)
- Auth JWT ([golang-jwt/jwt](https://github.com/golang-jwt/jwt)): access token de curta duração + refresh token mais longo, rotacionado e persistido no servidor para permitir revogação (`internal/models/refresh_token.go`)
- `golang.org/x/crypto/bcrypt`: hash de senha
- Middleware [`limiter`](https://docs.gofiber.io/api/middleware/limiter) embutido do Fiber: rate limiting (5 req/min/IP em login/registro), sem dependência extra
- Email via `net/smtp`, com fallback no console em dev (`internal/mail`): sem setup necessário para testar o fluxo de auth localmente
- `log/slog` (stdlib): logging estruturado, texto legível em dev, JSON em prod
- Formato consistente `{"error": {"code", "message", "details"}}` em todo endpoint (`internal/apierror`)
- [go-playground/validator](https://github.com/go-playground/validator): validação de request via struct tags, nomes de campo JSON nos detalhes de erro
- `testing` do Go + [testify](https://github.com/stretchr/testify): testes unitários + integração contra um Postgres real (via `fiber.App.Test`, sem mock de banco)
- [golangci-lint](https://golangci-lint.run): meta-linter; hook de pre-commit nativo do git (`gofmt` + `golangci-lint`), sem precisar de tooling de outra linguagem
- Docker + docker-compose: build multi-stage, binário estático `CGO_ENABLED=0` numa imagem de runtime **distroless non-root**
- GitHub Actions CI: build/lint/vet/testes contra um container de serviço Postgres real, build da imagem Docker
- `/health` para o healthcheck do Docker (mais um binário `healthcheck` standalone, já que distroless não tem shell/curl)

## Estrutura do projeto

```
cmd/
  api/main.go                 # entrypoint: config, migrate, connect, server.New, listen
  healthcheck/main.go          # binário standalone para o HEALTHCHECK do Dockerfile
internal/
  config/                        # config baseada em env, com defaults sensatos
  server/                         # server.New — conecta toda rota; compartilhado por main.go e testes
  db/                              # conexão GORM + migrations do golang-migrate embutidas
    migrations/                     # *.up.sql / *.down.sql — commitar
  models/                           # User, RefreshToken, Note (GORM)
  apierror/                         # ApiError + ErrorHandler do Fiber → formato {"error": {...}}, bind de validação
  auth/                             # JWT, bcrypt, middleware RequireAuth/RequireAdmin, handlers de /auth/*
  account/                          # handlers de /account/*
  admin/                            # handlers de /admin/users, /admin/notes
  diagnostics/                      # QueryCounter (plugin GORM só de teste, guarda de N+1)
  notes/                            # handlers de /api/notes/* (CRUD de referência)
  mail/                             # envio SMTP, fallback console em dev
```

## Começando (Docker — recomendado)

```bash
cp .env.example .env
# gere um secret de verdade e coloque em .env como JWT_SECRET
openssl rand -base64 32

docker compose up -d --build
```

App: [http://localhost:8083](http://localhost:8083). O Postgres é exposto na porta `5460` do host por padrão (não `5432`, para evitar conflito com um Postgres local). Migrations rodam automaticamente ao subir o container.

## Começando (sem Docker)

Precisa de uma instância Postgres e do [Go](https://go.dev/doc/install) 1.25+.

```bash
cp .env.example .env   # aponte DATABASE_URL para seu próprio Postgres
go run ./cmd/api
```

## Variáveis de ambiente

Veja `.env.example` para a lista completa e comentada.

| Variável                           | Obrigatória | Propósito                                                          |
| ------------------------------------ | ----------- | ---------------------------------------------------------------------- |
| `DATABASE_URL`                       | sim         | String de conexão do Postgres                                          |
| `JWT_SECRET`                         | sim         | ≥32 caracteres; assinatura dos tokens de access/refresh                 |
| `DB_HOST_PORT/NAME/USER/PASSWORD`    | só Docker   | Padrões do `docker-compose.yml`, usados para compor `DATABASE_URL`      |
| `ENVIRONMENT`                        | não         | `dev` (logs legíveis) ou qualquer outro valor (logs JSON); padrão `dev` |
| `LOG_LEVEL`                          | não         | `debug` ou qualquer outro valor (info); padrão `info`                   |
| `MAIL_HOST`/`MAIL_PORT`/`MAIL_USERNAME`/`MAIL_PASSWORD` | não | Envia email real via SMTP; sem `MAIL_HOST`, os emails são logados no console |

`internal/config/config.go` carrega essas variáveis com defaults sensatos. Nada quebra por uma variável faltando, mas `JWT_SECRET` sempre deve ser sobrescrito fora do dev local.

## Auth

`internal/auth` (todos `/auth/*`, públicos):

- `POST /auth/register` — 201, corpo vazio. 409 se o email já existe.
- `GET /auth/verify-email?token=...` — 200 vazio. 404 token inválido, 409 expirado.
- `POST /auth/login` — 200, `{accessToken, refreshToken}`. 401 credenciais inválidas ou email não verificado.
- `POST /auth/refresh` — rotaciona o refresh token (o antigo é apagado, um novo é emitido). 401 se inválido/expirado/revogado.
- `POST /auth/logout` — 200, idempotente.
- `POST /auth/forgot-password` — sempre 200 (sem vazamento de enumeração de usuário).
- `POST /auth/reset-password` — 200. 404/409 igual ao verify-email.
- `PATCH /account/password`, `DELETE /account` (`internal/account`) — autenticados, `Authorization: Bearer <accessToken>`.
- Rate limit: 5 tentativas de registro/login por 60s por IP (middleware `limiter` do Fiber, ver `internal/auth/routes.go`).
- `auth.RequireAuth` (`internal/auth/middleware.go`) é o único middleware usado por toda rota protegida: decodifica e valida o bearer token, sem duplicação por rota. `auth.RequireAdmin` adiciona uma checagem de role em cima.

**Refresh tokens são persistidos.** Diferente de um esquema JWT puramente stateless, o `jti` de cada refresh token fica salvo na tabela `refresh_tokens` para permitir revogação ou rotação no servidor. Logout ou refresh apaga a linha antiga, então um refresh token roubado não pode ser reaproveitado depois da rotação.

## Papéis (roles)

Tipo `Role` (`USER` | `ADMIN`, padrão `USER`) em `User.Role`. Nunca confie num role vindo do corpo da requisição. `GET /admin/users` (`internal/admin`) é o endpoint de referência admin-only, protegido por `auth.RequireAdmin`. Sem auto-promoção: altere direto no banco para testar localmente: `UPDATE users SET role = 'ADMIN' WHERE email = '...';`.

## Formato de erro

Toda resposta de erro — validação, auth, not-found, não tratado — tem o mesmo envelope, produzido por `apierror.Handler` (o `ErrorHandler` global do Fiber):

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "Invalid request body", "details": ["email: failed on the 'email' rule"] } }
```

`code` é um de `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `RATE_LIMITED`, `INTERNAL_ERROR`, batendo de propósito com o formato usado pelo resto da família de backends, para um template de front conseguir trocar de backend com o mínimo de mudança no tratamento de erro.

## Banco de dados

Schema fica em `internal/db/migrations/` (SQL puro, formato golang-migrate) e `internal/models/` (structs GORM espelhando ele, usados para query e não para gerar o schema; este template **não** depende de `AutoMigrate` em produção). Depois de mudar o schema:

```bash
migrate create -ext sql -dir internal/db/migrations -seq add_something
# edite os *.up.sql / *.down.sql gerados, depois atualize internal/models/ para bater
```

(Precisa da [CLI `migrate`](https://github.com/golang-migrate/migrate#cli-usage) localmente só para gerar os nomes de arquivo de migration novos. O app em si aplica as migrations via a biblioteca embutida, sem precisar de CLI em runtime.)

Toda foreign key para `users` usa `ON DELETE CASCADE` desde a primeira migration, então excluir uma conta limpa automaticamente suas notas e refresh tokens (ver [Notas de design](#notas-de-design-e-pegadinhas)).

## Exemplo de recurso CRUD

`/api/notes` (`internal/notes`) é uma implementação de referência completa: um request validado → um model GORM pertencente ao usuário autenticado → uma resposta JSON com nomes de campo em camelCase batendo com a convenção do resto da família. Copie esse formato para sua primeira feature de verdade, depois apague `internal/notes` (e remova a tabela `notes` via uma migration nova) quando não precisar mais da referência.

## Testes

- **Unitário** (`go test ./internal/auth/...`): hash de senha e roundtrip de JWT, sem banco — `internal/auth/auth_test.go`.
- **Integração** (`go test ./internal/server/...`): `internal/server/server_test.go` percorre o fluxo completo registro → verificação → login → CRUD de notas → refresh → exclusão de conta através de `fiber.App.Test()` (in-process, sem listener de rede real) contra um Postgres real, sem mock de banco. Defina `TEST_DATABASE_URL` para apontar para um banco específico (usa `DATABASE_URL`/seu padrão como fallback).
- **Guarda de contagem de queries (N+1)** (`go test ./internal/admin/...`): `internal/admin/query_count_test.go` semeia uma quantidade variável de notas, registra `diagnostics.QueryCounter` (um plugin GORM que conta cada statement SQL executado) na conexão, e garante que `admin.NotesWithOwners` — a função por trás de `GET /admin/notes` — sempre roda exatamente 1 query SQL, tanto com 4 quanto com 8 notas. `NotesWithOwners` usa `Joins("Owner")` do GORM (um único join SQL) em vez de carregar o dono de cada nota numa query separada; se alguém trocar isso por uma busca por nota, esse teste quebra antes de chegar em produção.
- O CI sobe um container de serviço Postgres e roda a suíte inteira (`go test ./...`) contra ele.

## CI/CD

`.github/workflows/ci.yml` roda dois jobs a cada push/PR:

1. **build**: `go build`, `gofmt -l` (falha em arquivos não formatados), `golangci-lint run`, `go vet`, `go test ./...`, tudo contra um container de serviço Postgres real
2. **docker**: builda a imagem Docker de produção (`docker/build-push-action`, sem push) para pegar quebra no Dockerfile cedo

Dependabot (`.github/dependabot.yml`) checa módulos Go, GitHub Actions e o Dockerfile semanalmente.

## Docker

- `Dockerfile`: multi-stage (`build` → `runtime`). O estágio de build compila com `CGO_ENABLED=0` para um binário totalmente estático; o estágio de runtime é `gcr.io/distroless/static:nonroot`, sem shell, sem gerenciador de pacotes, com cerca de 2MB de base, rodando como usuário non-root por padrão.
- Como o distroless não tem `curl`/`wget`/shell para um `HEALTHCHECK CMD`, `cmd/healthcheck` é um segundo binário Go pequeno (um GET HTTP simples para `/health`) compilado e copiado para dentro da mesma imagem.
- `docker-compose.yml`: `db` (Postgres 17, healthcheck via `pg_isready`, porta `5460` do host por padrão) e `app` (buildado do Dockerfile, healthcheck via o binário `/healthcheck`, espera o `db` ficar saudável).

## Scripts

| Comando                              | Propósito                          |
| --------------------------------------- | ------------------------------------ |
| `go run ./cmd/api`                       | Inicia o dev server                  |
| `go build ./...`                          | Builda tudo                          |
| `go test ./...`                            | Roda os testes                       |
| `gofmt -w .`                                | Formata                              |
| `golangci-lint run ./...`                    | Lint                                  |
| `git config core.hooksPath githooks`          | Uma vez: habilita o hook de pre-commit |
| `docker compose up -d --build`                 | Builda e sobe app + Postgres         |
| `docker compose down`                           | Para                                  |

## Usando como template

1. Clique em "Use this template" no GitHub
2. `go mod edit -module github.com/<voce>/<repo>` e atualize todo import path (`grep -rl obrenoalvim/back-template-go .` para achar todos), e atualize este README
3. `cp .env.example .env`, defina um `JWT_SECRET` de verdade
4. `git config core.hooksPath githooks` (uma vez, por clone; o hook deste repo não é instalado automaticamente do jeito que `npm install` dispara o Husky)
5. `docker compose up -d --build` (ou o caminho sem Docker acima)
6. Apague `internal/notes` depois de copiar o padrão para sua primeira feature

## Notas de design e pegadinhas

- **`ON DELETE CASCADE` em toda FK para `users`, desde a primeira migration.** Escrito assim desde o início porque o `back-template-fastapi` (construído mais cedo na mesma sessão) foi publicado sem isso, e `DELETE /account` quebrava com um `ForeignKeyViolationError` de verdade assim que um usuário tinha alguma nota. Excluir um usuário precisa cascatear no nível do banco; "apagar filhos primeiro" na aplicação é mais uma coisa para esquecer quando uma tabela filha nova aparecer depois.
- **`fiber.Ctx.SendStatus` não é "seta status, corpo vazio."** Ele preenche o corpo com o texto do status (`"Created"`, `"OK"`, ...) sempre que nada mais foi escrito: um corpo literal de 7 bytes no que o resto da família de backends retorna como genuinamente vazio. `apierror.Empty(c, status)` (`c.Status(status).Send(nil)`) é usado em todo endpoint que deveria ter resposta verdadeiramente vazia.
- **Um middleware de logging que lê o status antes do error handler escrever ele sempre loga 200.** O `ErrorHandler` configurado do Fiber só roda depois que toda a cadeia de middleware (incluindo um middleware de logging envolvendo tudo com `c.Next()`) já devolveu o controle para o dispatcher externo do Fiber, então um `status := c.Response().StatusCode()` ingênuo logo depois do `c.Next()` lê o status padrão pré-erro. Toda requisição 4xx/5xx era logada como 200 até o `requestLogger` (`internal/server/server.go`) ser mudado para chamar `apierror.Handler` ele mesmo quando `c.Next()` retorna um erro não-nulo, antes de ler o status.
- **`go-playground/validator` reporta nomes de campo do Go, não do JSON, a menos que você diga para não fazer isso.** `fe.Field()` retorna `"Email"` (o campo da struct) por padrão, não `"email"` (a chave JSON), inconsistente com todo outro backend da família, que reportam o nome de campo no formato de wire. Corrigido via `validate.RegisterTagNameFunc` em `internal/apierror/bind.go`.
- **Migrations são embutidas (`go:embed`), não lidas do disco em runtime.** A imagem de runtime distroless não tem acesso a filesystem para um diretório `migrations/` enviado separadamente, então `internal/db/db.go` embute `internal/db/migrations/*.sql` direto no binário compilado via `//go:embed all:migrations`. `go build` produz um artefato único e autocontido, sem arquivos externos para copiar para dentro da imagem Docker além do próprio binário.
