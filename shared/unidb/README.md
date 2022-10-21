# shared/uni-db

A special octopus specific database accessing framework aimed to completely eliminate need for database configuration other then just setting database server's host name in the source code or configuration. All code executing on developer's computer would perform exactly the same on the production database. This is achieved by unification of access via ssh tunnel and direct tcp connection.

During start-up phase `UniDB` tries to check if direct tcp connection is available, by probing configured or default (3306) port on target server. If direct connection is available, UniDB creates regular `sqlx` connection and sets up the rests.

In case direct tcp connection is unavailable `UniDB` brings up the ssh tunnel to the remote, using bundled ssh private key if it's not overridden by configuration.



# builder pattern

`UniDB` configuration follows the builder pattern due to the need of flexible options override, here is an example of full conjuration and start-up:

```go
//go:embed test_queries.sql
var queries embed.FS

func connection() {
	pk, _ := keys.GetAnsiblePrivateKey()               // it's the default key as well
	
	uniDB, err := NewUniDB().
		WithHost("root@my-server").
		WithDB("mydb").
		WithQueries(&queries).
		WithLogger(zerolog.Logger{}).                    // new logger with unidb marker
                                                         // is created by default
		WithSshKey(pk).
		WithSshKeyFile("/tmp/the-key").
		WithTCPTimeout(10 * time.Second).                // timeout to wait for tcp probe
                                                         // 3 seconds by default
  
		WithDBDriverArgs("timeout=30s&readTimeout=30s"). // these are defaults as well
		WithMaxIdleConns(10).
		WithMaxIdleConnTime(10 * time.Second).
		Connect()
}
```

