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
