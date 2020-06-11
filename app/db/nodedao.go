package db

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type NodeDao interface {
	CreateTree(root Node)
	Connect(first, second Node)
}

type NodeDaoNeoImpl struct{}

func (dao NodeDaoNeoImpl) CreateTree(root Node) {
	session, driver := connectToNeo()
	defer driver.Close()
	defer session.Close()

	_, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		query := "CREATE\n"
		count := 0

		createQuery(root, &count, &query)

		// trim final ,\n
		if len(query) > 0 {
			query = query[:len(query)-2]
		}

		_, err := transaction.Run(
			query,
			nil)

		return nil, err
	})
	if err != nil {
		panic(err)
	}
}

// wrapper for recursive function to create a query
func createQuery(node Node, count *int, query *string) {
	m := make(map[string]int)
	createQueryRecur(node, count, query, &m)
}

// recrusive call that dives down through children, creating the create statements for each node,
// then going back up to create the relationships
func createQueryRecur(node Node, count *int, query *string, seenNodes *map[string]int) int {
	// remember the what the count was when this iteration started, so we can reference this node in the query
	var currCount int

	if realNode, ok := node.(*StatementNode); ok {
		fmt.Printf("we are line number %v", realNode.LineNumber)
	}

	// use the properties as a key; they include file/line number, so they are unique
	nodeKey := node.GetProperties()

	// check if we have seen this node or not
	if storedCount, ok := (*seenNodes)[nodeKey]; ok {
		// we've seen it, get the number so we can make the relationship, but don't process any children
		currCount = storedCount
	} else {
		// haven't seen it, create it and add it to the map
		currCount = *count
		*query += fmt.Sprintf("(n%v%v %v),\n", currCount, node.GetNodeType(), node.GetProperties())
		(*seenNodes)[nodeKey] = currCount

		for child, relationshipProps := range node.GetChildren() {
			if realNode, ok := node.(*StatementNode); ok {
				fmt.Printf("we are line number %v", realNode.LineNumber)
			}
			if child != nil {
				*count = *count + 1
				childCount := createQueryRecur(child, count, query, seenNodes)
				*query += fmt.Sprintf("(n%v)-[:FLOWSTO %v]->(n%v),\n", currCount, relationshipProps, childCount)
			}
		}
	}

	return currCount
}

func (dao NodeDaoNeoImpl) Connect(first, second Node) (string, error) {
	session, driver := connectToNeo()
	defer session.Close()
	defer driver.Close()

	//Connect
	if first.GetNodeType() == ":FUNCTIONCALL" {
		response, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			res, err := transaction.Run(
				//query for getting nodes from db
				//and adding relationship to connect the two graphs
				`MATCH (a:Node), (b:Node) WHERE a.filename = $callerFile 
				AND b.filename = $calleeFile AND a.line = $callerLine 
				AND b.line = $calleeLine 
				CREATE e = (a)-[r:FLOWSTO]->(b) 
				RETURN e`,
				map[string]interface{}{"callerFile": first.GetFilename(), "calleeFile": second.GetFilename(),
					"callerLine": first.GetLineNumber(), "calleeLine": second.GetLineNumber()})
			if err != nil {
				return nil, err
			}

			if res.Next() {
				return res.Record().GetByIndex(0), nil
			}

			return nil, res.Err()
		})
		if err != nil {
			return "", err
		}
		return response.(string), nil
	} else {
		return "", nil
	}
}
