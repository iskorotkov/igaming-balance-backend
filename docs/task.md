# Golang Balance Operations Processing System - Technical Assessment

## Objective
The objective of this test task is to assess the candidate's proficiency in Golang programming, Docker containerization, and database integration using Postgres. The task involves implementing a Golang application that processes and saves incoming requests via gRPC, representing part of a balance operations processing system.

## Requirements

### 1. Git Repository
- The code should be hosted in a Git repository service of the candidate's choice (e.g., GitHub, GitLab, Bitbucket).

### 2. Docker Containerization
- The application should be containerized using Docker, allowing for easy deployment and scalability.

### 3. README File
- The README file should contain comprehensive instructions for running the application out of the box.
- This includes steps for building Docker containers, running the application, and any additional configuration or setup required.

## Technologies
- **GoLang** for application development
- **Postgres** for database integration

## Task Description

### 1. Processing and Saving Incoming Requests via gRPC
Requests represent balance operations and include fields such as source, state, amount, and transaction ID:

- **Source** can be one of three types: `game`, `payment`, or `service`
- **State** can be one of two types: `deposit` and `withdraw` with corresponding actions to increase or decrease the balance
- Each request, identified by a `tx_id`, must be processed only once

### 2. Post-Processing Part
- Periodically, every N minutes, the application cancels the 10 latest odd operations (1, 3, 5, 7, 9...)
- Cancelled operations should not be processed again
- The balance should be corrected accordingly after canceling the operations

## Data Models

### JSON Model Representation
```json
{
    "source": "client",
    "state": "deposit",
    "amount": "10.15",
    "tx_id": "some generated identificator"
}
```

### gRPC Message Representation
```protobuf
message Request {
    string source = 1;
    string state = 2;
    string amount = 3;
    string tx_id = 4;
}
```

## Additional Considerations
- A negative balance state is not permitted
- The decision regarding database architecture and table structure is on you
