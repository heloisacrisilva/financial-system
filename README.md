# Financial System API

API REST em Go para processamento de transações financeiras com contas, clientes, histórico, `reference_id` gerado pela API, controle de concorrência por lock pessimista e eventos assíncronos via outbox.

## Stack

- Go 1.25
- Gin para HTTP
- GORM para persistência
- PostgreSQL
- Docker Compose para subir o banco local
- OpenAPI/Swagger para documentação da API

## Como Rodar

### 1. Subir o PostgreSQL

```bash
docker-compose up -d
```

O banco sobe em `localhost:5435`.

### 2. Configurar variáveis de ambiente

O projeto espera:

```env
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=api_financial_system
DB_PORT=5435
DB_HOST=localhost
APP_PORT=8080
```

O arquivo `.env` local já contém a configuração do banco. `APP_PORT` é opcional; se não informado, a API usa `8080`.

### 3. Executar a aplicação

```bash
go run main.go
```

A API ficará disponível em:

```text
http://localhost:8080
```

## Testes

O projeto conta com testes unitários automatizados para a camada de HTTP (`handlers`), cobrindo fluxos de sucesso, falhas de validação de payload e regras de negócio. 

Os testes utilizam a técnica de substituição de funções por variáveis locais (Mocking de escopo síncrono), o que garante o isolamento da infraestrutura (os testes rodam instantaneamente sem precisar que o PostgreSQL esteja de pé).

Para rodar todos os testes do projeto com o log detalhado de cada cenário:

```bash
go test ./... -v
```

## Swagger / OpenAPI

Com a aplicação rodando:

- Swagger UI: http://localhost:8080/swagger
- OpenAPI YAML: http://localhost:8080/openapi.yaml

O contrato também está versionado em:

```text
api/docs/openapi.yaml
```

## Endpoints

### Health Check

```http
GET /health
```

### Contas

```http
POST /api/accounts/create
GET  /api/accounts/{id}
```

### Clientes

```http
GET /api/clients/{id}
```

### Operações

```http
POST /api/operations/{type}
GET  /api/operations/{id}
```

Tipos aceitos em `{type}`:

- `credit`
- `debit`
- `reserve`
- `capture`
- `transfer`
- `reversal`

## Exemplos de Uso

### Health Check

```bash
curl http://localhost:8080/health
```

Resposta esperada:

```json
{
  "status": "ok",
  "timestamp": "2026-07-13T10:00:00Z",
  "checks": {
    "api": "ok",
    "database": "ok"
  }
}
```

### Criar Conta

```bash
curl -X POST http://localhost:8080/api/accounts/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Maria Silva",
    "email": "maria@example.com",
    "cpf": "12345678901"
  }'
```

### Creditar Saldo

```bash
curl -X POST http://localhost:8080/api/operations/credit \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 100000,
    "currency": "BRL"
  }'
```

A resposta inclui `reference_id`, que pode ser usado depois como `original_reference_id` em um estorno.

### Debitar Saldo

```bash
curl -X POST http://localhost:8080/api/operations/debit \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 20000,
    "currency": "BRL"
  }'
```

### Reservar Saldo

```bash
curl -X POST http://localhost:8080/api/operations/reserve \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 30000,
    "currency": "BRL"
  }'
```

### Capturar Reserva

```bash
curl -X POST http://localhost:8080/api/operations/capture \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 30000,
    "currency": "BRL"
  }'
```

### Transferir Entre Contas

```bash
curl -X POST http://localhost:8080/api/operations/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "account_dest_id": 2,
    "value": 50000,
    "currency": "BRL"
  }'
```

### Estornar Operação

```bash
curl -X POST http://localhost:8080/api/operations/reversal \
  -H "Content-Type: application/json" \
  -d '{
    "original_reference_id": "reference-id-retornado-na-operação-original"
  }'
```

### Consultar Histórico Por ID

```bash
curl http://localhost:8080/api/operations/1
```

## Regras de Negócio Implementadas

- Crédito adiciona valor ao saldo disponível.
- Débito remove valor do saldo disponível e considera limite de crédito.
- Reserva move valor do saldo disponível para saldo reservado.
- Captura confirma uma reserva conforme regra do projeto, retirando do reservado e refletindo no saldo disponível.
- Transferência movimenta valor entre duas contas.
- Estorno reverte créditos, débitos e transferências.
- Contas precisam estar ativas para operar.
- Moeda da operação deve bater com a moeda da conta quando informada.
- A API gera um `reference_id` único para cada operação.
- Operações financeiras são atômicas com transação de banco.
- Operações na mesma conta usam lock pessimista (`SELECT FOR UPDATE`) para consistência em concorrência.

## Idempotência

A idempotência é controlada pela tabela `operation_references`, que possui `reference_id` único. O valor é gerado pela API quando a operação é recebida e retornado no payload de sucesso.

Esse controle fica separado de `transaction_histories` porque uma única operação pode gerar mais de uma linha de histórico. Exemplo: uma transferência gera uma "perna" de débito e outra de crédito, ambas com o mesmo `reference_id`.

Para estornar uma operação, use o `reference_id` retornado pela operação original no campo `original_reference_id`.

## Eventos Assíncronos

O projeto usa o padrão outbox:

1. A operação financeira grava saldo/histórico dentro de uma transação.
2. Na mesma transação, grava um registro em `operation_events` com status `pending`.
3. Um worker em background busca eventos pendentes.
4. O worker publica o evento e marca como `published`.
5. Se falhar, marca como `failed` e agenda retry com backoff exponencial.

Hoje a publicação real é simulada por log no método `publishEvent`. A estrutura está preparada para substituir essa função por Kafka, RabbitMQ, SQS ou outro broker.

## Decisões Técnicas

- **Gin**: framework HTTP simples e direto para APIs REST.
- **GORM**: reduz boilerplate de persistência e suporta transações, locks e migração automática.
- **PostgreSQL**: banco relacional adequado para consistência transacional e controle de concorrência.
- **Outbox pattern**: evita perder eventos quando a operação de negócio foi persistida, mas a publicação externa falha.
- **Lock pessimista**: usado nas contas durante operações financeiras para evitar race condition de saldo.
- **Retry com backoff**: aplicado tanto no handler para falhas retryable quanto no dispatcher de eventos.

## Estrutura do Projeto

```text
api/
  config/        Configuração de banco e logger
  docs/          Contrato OpenAPI
  entities/      Modelos persistidos
  events/        Dispatcher em background dos eventos
  handler/       Handlers HTTP
  helpers/       Funções auxiliares
  repository/    Regras de persistência e operações financeiras
  router/        Rotas HTTP
main.go          Bootstrap da aplicação
docker-compose.yml
```

## Observabilidade

O projeto possui logs para:

- inicialização da aplicação;
- erros de validação;
- erros de banco;
- falhas de operações;
- publicação de eventos;
- falhas e retries no dispatcher de eventos.
- health check da API e conectividade com PostgreSQL.

## Limitações Conhecidas

- Ainda não há testes de integração automatizados que batam diretamente no banco de dados real (PostgreSQL) ou testes específicos para a camada de persistência (`repository`).
