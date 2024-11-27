## Usage

# Build
build the application with below command

```bash
go build .
```

# Backup projects with git repos
```bash
./projectsync -backup -dir /path/to/your/repos -config repos_config.json
```

# Restore projects with git repos
```bash
./projectsync -restore -config repos_config.json
```
