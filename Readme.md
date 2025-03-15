# Social App

## Getting Started

These instructions will help you set up and run the server on your local machine.

### Prerequisites

- Docker
- Docker Compose

### Running the Server

1. Clone the repository:

    ```sh
    git clone https://github.com/yourusername/social.git
    cd social
    ```

2. Build and run the Docker containers:

    ```sh
    docker-compose up --build
    ```

3. The server will be available at `http://localhost:8080`.

### API Endpoints

- `POST /user/register`: Register a new user.
- `POST /login`: Login with user credentials.
- `GET /user/get/{id}`: Get user details by ID.

### Environment Variables

The following environment variables can be set in the `docker-compose.yml` file:

- `DB_HOST`: Database host (default: `db`)
- `DB_PORT`: Database port (default: `5432`)
- `DB_USER`: Database user (default: `postgres`)
- `DB_PASSWORD`: Database password (default: `postgres`)
- `DB_NAME`: Database name (default: `social`)

### Example Requests

#### Register User

```sh
curl -X POST http://localhost:8080/user/register -H "Content-Type: application/json" -d '{
  "first_name": "Имя",
  "last_name": "Фамилия",
  "birthdate": "2000-01-01",
  "biography": "Хобби, интересы и т.п.",
  "city": "Москва",
  "password": "Секретная строка"
}'
```

#### Login

```sh
curl -X POST http://localhost:8080/login -H "Content-Type: application/json" -d '{
  "id": "user-id",
  "password": "Секретная строка"
}'
```

#### Get User by ID

```sh
curl -X GET http://localhost:8080/user/get/user-id
```

### License

This project is licensed under the MIT License.