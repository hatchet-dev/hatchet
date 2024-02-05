# Hatchet FastAPI Example

This is an example project demonstrating how to use Hatchet with FastAPI.

## Prerequisites

Before running this project, make sure you have the following:

1. Python 3.7 or higher installed on your machine.
2. Poetry package manager installed. You can install it by running `pip install poetry`.
3. Clone this repository to your local machine.

## Setup

1. Create a `.env` file in the project root directory and set the required environment variables. Refer to the documentation for the specific environment variables needed for your application.

2. Open a terminal and navigate to the project root directory.

3. Run the following command to install the project dependencies:

   ```shell
   poetry install
   ```

## Running the API

To start the FastAPI server, run the following command in the terminal:

```shell
poetry run api
```

## Running the Hatchet Worker

To start the Hatchet worker, run the following command in the terminal:

```shell
poetry run hatchet
```
