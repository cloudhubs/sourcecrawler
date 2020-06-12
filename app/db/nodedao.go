package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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
		fmt.Printf("we are line number %v\n", realNode.LineNumber)
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
				fmt.Printf("we are line number %v\n", realNode.LineNumber)
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

func (dao NodeDaoNeoImpl) FindNode(filename string, linenumber int) (Node, error) {
	session, driver := connectToNeo()
	defer driver.Close()
	defer session.Close()

	response, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err := transaction.Run(
			`
			MATCH (a:STATEMENT {filename: $file, linenumber: $line})
			RETURN a
			`,
			map[string]interface{}{"file": filename, "line": strconv.Itoa(linenumber)})
		if err != nil {
			return nil, err
		}
		if result.Next() {
			if node, ok := result.Record().GetByIndex(0).(neo4j.Node); ok {
				nodeFile := node.Props()["filename"].(string)
				nodeLine, _ := strconv.Atoi(node.Props()["linenumber"].(string))
				for _, v := range node.Labels() {
					switch v {
					case "FUNCTIONCALL":
						return &FunctionNode{nodeFile, nodeLine, node.Props()["function"].(string), *new(Node)}, nil
					case "CONDITIONAL":
						return &ConditionalNode{nodeFile, nodeLine, node.Props()["condition"].(string), *new(Node), *new(Node)}, nil
					default:
						if regex, ok := node.Props()["logregex"]; ok {
							return &StatementNode{nodeFile, nodeLine, regex.(string), *new(Node)}, nil
						}
						return &StatementNode{nodeFile, nodeLine, "", *new(Node)}, nil
					}
				}
			}
		}
		return nil, result.Err()
	})
	if err != nil {
		return nil, err
	}
	return response.(Node), nil

}

func (dao NodeDaoNeoImpl) Connect(first, second Node) (string, error) {
	session, driver := connectToNeo()
	defer session.Close()
	defer driver.Close()

	//Connect
	if strings.Contains(first.GetNodeType(), ":STATEMENT") {
		response, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			//find first node and child

			//find second node and the last node it its chain

			//connect first node to second node,

			//connect last node in second node's
			//chain to the child of the first node

			_, err := transaction.Run(
				//query for getting nodes from db
				//and adding relationship to connect the two graphs
				`MATCH (a:STATEMENT{filename: $firstFile, linenumber: $firstLine })-[toRemove:FLOWSTO]->(c:STATEMENT),
				(b:STATEMENT {filename: $secondFile, linenumber: $secondLine})-[*]->(d:STATEMENT) 
				WHERE NOT (d)-->() 
				MERGE e1 = (a)-[r1:FLOWSTO]->(b) 
				MERGE e2 = (d)-[r2:FLOWSTO]->(c) 
				DELETE toRemove
				`,
				map[string]interface{}{"firstFile": first.GetFilename(), "secondFile": second.GetFilename(),
					"firstLine": strconv.Itoa(first.GetLineNumber()), "secondLine": strconv.Itoa(second.GetLineNumber())})
			if err != nil {
				return nil, err
			}
			return "success", nil
		})
		if err != nil {
			return "", err
		}
		return response.(string), nil
	}
	return "", errors.New("Node is wrong type")
}
