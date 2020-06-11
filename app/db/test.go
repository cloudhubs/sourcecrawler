package db

// func helloWorld(uri, username, password string) (string, error) {
// 	var (
// 		err      error
// 		driver   neo4j.Driver
// 		session  neo4j.Session
// 		result   neo4j.Result
// 		greeting interface{}
// 	)
// 	useConsoleLogger := func(level neo4j.LogLevel) func(config *neo4j.Config) {
// 		return func(config *neo4j.Config) {
// 			config.Log = neo4j.ConsoleLogger(level)
// 			config.Encrypted = false
// 		}
// 	}
// 	driver, err = neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""), useConsoleLogger(neo4j.WARNING))
// 	if err != nil {
// 		return "", err
// 	}
// 	defer driver.Close()
// 	session, err = driver.Session(neo4j.AccessModeWrite)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer session.Close()
// 	greeting, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
// 		result, err = transaction.Run(
// 			"CREATE (a:Greeting) SET a.message = $message RETURN a.message + ', from node ' + id(a)",
// 			map[string]interface{}{"message": "hello, world"})
// 		if err != nil {
// 			return nil, err
// 		}
// 		if result.Next() {
// 			return result.Record().GetByIndex(0), nil
// 		}
// 		return nil, result.Err()
// 	})
// 	if err != nil {
// 		return "", err
// 	}
// 	return greeting.(string), nil
// }
