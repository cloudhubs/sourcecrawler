package db

import "github.com/neo4j/neo4j-go-driver/neo4j"

type NodeDao interface {
	CreateTree(root *Node)
	Connect(first, second *Node)
}

type NodeDaoNeoImpl struct{}

func (dao NodeDaoNeoImpl) CreateTree(root *Node) {
	session, driver := connect()
	defer driver.Close()
	defer session.Close()

	_, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		var currNode, parentNode *Node
		currNode = root
		parentNode = nil
		done := false

		for !done {

		}

		result, err = transaction.Run(
			"CREATE (a:Greeting) SET a.message = $message RETURN a.message + ', from node ' + id(a)",
			map[string]interface{}{"message": "hello, world"})
		if err != nil {
			return nil, err
		}
		if result.Next() {
			return result.Record().GetByIndex(0), nil
		}
		return nil, result.Err()
	})
	if err != nil {
		panic(err)
	}
	return greeting.(string), nil
}
