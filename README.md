# Octopus - A Base Framework for Monorepos

Octopus is an open-source framework written in Go, designed to provide a strong foundation for building monorepos. 
With Octopus, you can manage your monorepo faster, with unique libraries and a shorter time to production.

## Key Features

- All-in-one monorepo management solution
- Fast and efficient management of servers and infrastructure
- Easy to use configurations
- Simple building process with unique libraries
- Streamlined frontend development
- Built in Go for increased performance and stability

## Getting Started

Octopus is a base framework for building monorepos, rather than a tool for managing them. 
To get started with Octopus, follow these steps:

1. Fork the repository: visit the following URL to fork the repository: https://github.com/octopus-foundation/octopus/fork
2. Clone your forked repository:
    ```shell
    git clone https://github.com/<your-username>/octopus.git
    cd octopus
    ```
3. Use Octopus as a base framework to build your own monorepo solution.

## Keeping Up-to-Date with the Original Repository

As Octopus is an active project, new features and bug fixes may be added to the original repository. 
To keep your forked repository up-to-date, you can pull changes from the original repository.
1. Add the original repository as a remote to your local repository:
    ```shell
   git remote add upstream https://github.com/octopus-foundation/octopus.git
    ```
2. Fetch the latest changes from the original repository:
    ```shell
    git fetch upstream
    ```
3. Merge the changes into your local branch:
    ```shell
    git merge upstream/main
    ```
## Base Structure

Octopus is structured in a way that allows for easy management of the monorepo. 
The following directories make up the base structure of Octopus:

- `ansible`: Contains descriptions of all production and test servers, as well as the roles for each server.
- `appsconfigs`: A unified directory for application configurations.
- `build-tools`: Includes various docker files and build utilities.
- `parts`: Source code for the project, divided by areas of activity.
- `protobufs`: Protobuf files, divided by areas of activity.
- `shared`: Common libraries for all areas of activity.
- `vendor`: Vendoring for Golang.

Each of these directories serves a specific purpose and helps to keep the monorepo organized and manageable.

## Base Make Targets
Octopus provides a number of base Make targets that make it easy to build, test, and manage the monorepo. The following is a list of the available targets:

- `make protobuf`: Compiles the protobuf files.
- `make configs`: Generates the application configurations.
- `make binaries`: Builds all of the binary components.
- `make binary-only BIN_PATH=parts/smth/bin`: Builds only a specific binary component, specified by the `BIN_PATH` argument.
- `make checks`: Performs various checks on the source code and configuration files.
- `make tests`: Runs all of the tests for the project.
- `make full-tests`: Runs all of the tests, including the full integration tests.

Each of these targets serves a specific purpose and makes it easy to build and manage the monorepo. By using these targets, developers can quickly build, test, and deploy the components of their monorepo with ease.

## Contributing

Octopus is an open source project and contributions are welcome.

## License

Octopus is licensed under the [MIT License](./LICENSE).