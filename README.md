# Merkle Tree File Storage Solution

## Introduction
The developed application allows a client to upload files to a server, with integrity assured by a Merkle tree. 
The client saves the Merkle root hash locally for future file verification, while the proofs are stored in a PostgreSQL database on the server. 
The application is written in Go and uses MinIO for distributed object storage.

## Implementation Overview
The application consists of a client and server, which communicate via HTTP. The client streams files to the server, which computes the Merkle proofs and stores them in a PostgreSQL database. The client retains the Merkle root hash for future file validation.
Utilizing Go for its strong concurrency and networking, the application supports file uploads via streaming and integrity checks with Merkle proofs.

### Client
The client streams files in small chunks, optimizing memory usage. It computes the Merkle root hash in parallel and retains it for future validation.

### Server
The server manages uploads using goroutines and channels for high concurrency. It batch-processes file metadata and leverages MinIO for distributed object storage.

### Merkle Tree
A concurrent Merkle tree implementation offers significant performance gains (4-6x faster than sequential approaches), improving proof generation efficiency.

### Data Storage
PostgreSQL stores the proofs, enabling robust data management and integrity checking without persisting the entire tree, minimizing storage demands.

## Manual Testing
The application can be tested manually using the following steps:

1. **Build the application**: `go build .`
2. **Start the server**: `docker compose up`
3. **Upload files**: `./fileserver upload ./testdata http://localhost:8080/file`
4. **Compute Merkle root hash**: `./fileserver merkle ./testdata`
5. **Download file**: `./fileserver download 1 http://localhost:8080/file`
6. **Verify file**: `./fileserver verify ./testdata/1 ./1.proof ./merkle_root`

## Shortcomings and Future Improvements

### Shortcomings
1. **Testing**: The application's test coverage is incomplete, especially for the Merkle tree.
2. **RAM Usage**: While the Merkle tree padding doesn't affect persistent storage, it can increase the application's memory footprint during operation.
3. **Code Quality**: Refactoring could improve code readability and maintainability.
4. **User Data Segregation**: The system currently does not differentiate between users' data.
5. **Networking**: The networking setup could be enhanced for more seamless multi-machine deployment.

### Improvements
1. **Enhanced Testing**: Develop a comprehensive suite of unit and integration tests.
2. **Memory Optimization**: Investigate more efficient memory usage, particularly in the file streaming and Merkle tree operations.
3. **Refactoring**: Clean up the codebase to ensure maintainability.
4. **Data Segregation**: Implement user-based data segregation for improved security and organization.
5. **Networking**: Develop a more robust networking setup for multi-machine deployment.

## Conclusion
The application stands as a robust platform for file uploads with integrity checks via a Merkle tree. Future improvements could refine memory usage, increase test coverage, and ensure the application scales effectively in distributed environments.


