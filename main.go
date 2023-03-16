package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/pkg/sftp"
	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	vmName  string
	IP      string
	dnsName string
}

type Node struct {
	class      string
	name       string
	cond       string
	properties map[string]any
}

type Relationship struct {
	left       *Node
	class      string
	right      *Node
	properties map[string]any
}

type SSHClient struct {
	connection *ssh.Client
	sftp       *sftp.Client
	config     *ssh.ClientConfig
	ip         string
	port       string
	protocol   string
}

func (n Node) Add(session neo4j.SessionWithContext, ctx context.Context) (any, error) {
	return session.ExecuteWrite(ctx, n.getAddTransaction(ctx))
}

func (n Node) getAddTransaction(ctx context.Context) neo4j.ManagedTransactionWork {
	currentStatement := "CREATE (a:" + n.class + " {name: $name"
	if n.properties != nil {
		for field := range n.properties {
			currentStatement += ", " + field + ": $" + field
		}
	} else {
		n.properties = map[string]any{"name": ""}
	}
	currentStatement += "}) return a"

	n.properties["name"] = n.name
	return func(tx neo4j.ManagedTransaction) (any, error) {
		var result, err = tx.Run(ctx, currentStatement, n.properties)
		if err != nil {
			return nil, err
		}

		return result.Consume(ctx)
	}
}

func (n Node) Update(session neo4j.SessionWithContext, ctx context.Context) (any, error) {
	return session.ExecuteWrite(ctx, n.getUpdateTransaction(ctx))
}

func (n Node) getUpdateTransaction(ctx context.Context) neo4j.ManagedTransactionWork {
	currentStatement := "MATCH (a:" + n.class + " {name: $name}) SET "
	first := true
	if n.properties != nil {
		for field := range n.properties {
			if !first {
				currentStatement += ", "
			}
			currentStatement += "a." + field + "= $" + field
			first = false
		}
	} else {
		n.properties = map[string]any{"name": ""}
	}
	currentStatement += " return a"

	n.properties["name"] = n.name
	return func(tx neo4j.ManagedTransaction) (any, error) {
		var result, err = tx.Run(ctx, currentStatement, n.properties)
		if err != nil {
			return nil, err
		}

		return result.Consume(ctx)
	}
}

func (n Node) Exists(session neo4j.SessionWithContext, ctx context.Context) (bool, error) {
	count, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		fieldMap := map[string]any{}
		if n.cond == "" {
			n.cond = "name: $name"
			fieldMap["name"] = n.name
		}
		result, err := tx.Run(ctx, "MATCH (a:"+n.class+" {"+n.cond+"}) RETURN count(a) as count", fieldMap)
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			count, found := result.Record().Get("count")
			if found {
				return count, nil
			}
		}

		return 0, nil
	})
	if err != nil {
		return false, err
	}

	if count.(int64) > 0 {
		return true, nil
	}

	return false, nil
}

func (r Relationship) Add(session neo4j.SessionWithContext, ctx context.Context) (any, error) {
	return session.ExecuteWrite(ctx, r.getAddTransaction(ctx))
}

func (r Relationship) getAddTransaction(ctx context.Context) neo4j.ManagedTransactionWork {
	fieldMap := map[string]any{}
	if r.left.cond == "" {
		r.left.cond = "name: $nameL"
	}
	if r.right.cond == "" {
		r.right.cond = "name: $nameR"
		fieldMap["nameR"] = r.right.name
	}
	currentStatement := "MATCH (a:" + r.left.class + " {" + r.left.cond + "}), (b:" + r.right.class + " {" + r.right.cond + "}) CREATE (a)-[r:" + r.class
	if r.properties != nil {
		first := true
		for field := range r.properties {
			if first {
				currentStatement += " {"
				first = false
			} else {
				currentStatement += ", "
			}
			currentStatement += field + ": $" + field
			fieldMap[field] = r.properties[field]
		}
		if !first {
			currentStatement += "}"
		}
	}
	currentStatement += "]->(b) return a"

	fieldMap["nameL"] = r.left.name
	fieldMap["nameR"] = r.right.name

	//log.Println(currentStatement)
	//log.Println(fieldMap)

	return func(tx neo4j.ManagedTransaction) (any, error) {
		var result, err = tx.Run(ctx, currentStatement, fieldMap)
		if err != nil {
			return nil, err
		}

		return result.Consume(ctx)
	}
}

func (r Relationship) Delete(session neo4j.SessionWithContext, ctx context.Context) (any, error) {
	return session.ExecuteWrite(ctx, r.getDeleteTransaction(ctx))
}

func (r Relationship) getDeleteTransaction(ctx context.Context) neo4j.ManagedTransactionWork {
	return func(tx neo4j.ManagedTransaction) (any, error) {
		var result, err = tx.Run(ctx, "MATCH (a:"+r.left.class+" {name: $nameL})-[c:"+r.class+"]->(b:"+r.right.class+" {name: $nameR}) DELETE c return a", map[string]any{"nameL": r.left.name, "nameR": r.right.name})
		if err != nil {
			return nil, err
		}

		return result.Consume(ctx)
	}
}

func (r Relationship) Exists(session neo4j.SessionWithContext, ctx context.Context) (bool, error) {
	count, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		fieldMap := map[string]any{}
		if r.left.cond == "" {
			r.left.cond = "name: $nameL"
			fieldMap["nameL"] = r.left.name
		}
		if r.right.cond == "" {
			r.right.cond = "name: $nameR"
			fieldMap["nameR"] = r.right.name
		}
		result, err := tx.Run(ctx, "MATCH (a:"+r.left.class+" {"+r.left.cond+"})-[c:"+r.class+"]->(b:"+r.right.class+"{"+r.right.cond+"}) RETURN count(a) as count", fieldMap)
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			count, found := result.Record().Get("count")
			if found {
				return count, nil
			}
		}

		return 0, nil
	})
	if err != nil {
		return false, err
	}

	if count.(int64) > 0 {
		return true, nil
	}

	return false, nil
}

func (client SSHClient) executeScript(script string) (string, error) {
	tempFile := ksuid.New()
	dstFile, err := client.sftp.Create("/tmp/" + tempFile.String())
	if err != nil {
		log.Println("SFTP: Can't create remote file /tmp/" + tempFile.String() + " :" + err.Error())
		return "", err
	}
	defer dstFile.Close()

	srcFile, err := os.Open(script)
	if err != nil {
		log.Println("SFTP: Can't open file " + script + " :" + err.Error())
		return "", err
	}
	defer srcFile.Close()

	log.Println("SFTP: Copy script to server")
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		log.Println("SFTP: An error occured while copying script:" + err.Error())
		return "", err
	}
	srcFile.Close()
	dstFile.Close()

	session, err := client.connection.NewSession()
	if err != nil {
		log.Println("SSH: Can't create session :" + err.Error())
		return "", err
	}
	defer session.Close()

	log.Println("Execute script")
	out, err := session.CombinedOutput("chmod +x /tmp/" + tempFile.String() + ";/tmp/" + tempFile.String() + ";rm /tmp/" + tempFile.String())
	if err != nil {
		log.Println("Can't execute script /tmp/" + tempFile.String() + " :" + err.Error())
		return "", err
	}

	return string(out), nil
}

func (s *SSHClient) Connect() error {
	var err error
	log.Println("SSH: Connecting to server")
	sshc, err := ssh.Dial(s.protocol, s.ip+":"+s.port, s.config)
	if err != nil {
		return err
	}
	s.connection = sshc

	sftpc, err := sftp.NewClient(s.connection)
	if err != nil {
		return err
	}
	s.sftp = sftpc

	return nil
}

func main() {
	logFileName := os.Args[1]
	user := os.Args[2]
	pass := os.Args[3]
	discoveryListFileName := os.Args[4]
	neoHost := os.Args[5]
	neoPort := os.Args[6]

	now := time.Now()
	logFile, err := os.OpenFile(logFileName+"_"+strconv.Itoa(now.Year())+strconv.Itoa(now.YearDay())+strconv.Itoa(now.Hour())+strconv.Itoa(now.Minute())+strconv.Itoa(now.Second())+".log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(io.MultiWriter(logFile, os.Stdout))

	f, err := os.Open(discoveryListFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var discoveryList []Server
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var currentServer Server
		line := scanner.Text()
		lineParts := strings.Split(line, ",")

		currentServer.vmName = lineParts[0]
		currentServer.IP = lineParts[1]
		currentServer.dnsName = lineParts[2]

		discoveryList = append(discoveryList, currentServer)
	}

	dbUri := "neo4j://" + neoHost + ":" + neoPort

	log.Println("Connecting to Neo4j at " + dbUri)
	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth("neo4j", ".2p5Rxpn201", ""))
	if err != nil {
		log.Fatal("Can't connect to Neo4j database " + dbUri + ": " + err.Error())
	}
	ctx := context.Background()
	defer driver.Close(ctx)

	neoSession := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neoSession.Close(ctx)

	for _, currentServer := range discoveryList {
		server := new(Node)
		server.class = "Server"
		server.name = currentServer.vmName
		server.cond = "ip: '" + currentServer.IP + "'"
		server.properties = map[string]any{
			"ip": currentServer.IP,
		}

		//log.Println(server)

		log.Println("--- Start discovery of server " + currentServer.vmName + "(" + currentServer.IP + ")")
		found, err := server.Exists(neoSession, ctx)
		if err != nil {
			log.Fatal(err)
		}

		if !found {
			log.Println("Add server to Neo4j")
			_, err := server.Add(neoSession, ctx)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Println("Update server to Neo4j")
			_, err := server.Update(neoSession, ctx)
			if err != nil {
				log.Fatal(err)
			}
		}

		sshClient := new(SSHClient)

		sshClient.config = &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{ssh.Password(pass)},
		}

		sshClient.config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		sshClient.ip = currentServer.IP
		sshClient.port = "22"
		sshClient.protocol = "tcp"

		err = sshClient.Connect()
		if err != nil {
			log.Println(err)
			continue
		}

		dirList, _ := os.ReadDir("./plugins")

		for _, dir := range dirList {
			log.Println("Load plugin " + dir.Name())
			viper.SetConfigType("json")
			viper.SetConfigFile("./plugins/" + dir.Name())
			viper.ReadInConfig()

			pluginType := viper.GetString("type")
			pluginScript := viper.GetString("script")

			out, err := sshClient.executeScript(pluginScript)
			if err != nil {
				log.Println(err)
				continue
			}

			switch pluginType {
			case "properties":
				//pluginOutputFormat := viper.GetString("output_format")
				pluginParams := viper.GetStringMap("node_params")
				cols_regexp := regexp.MustCompile(`\$(\d+)`)

				lines := strings.Split(string(out), "\n")
				for index := range lines {
					if lines[index] == "" {
						log.Println("Skip " + lines[index])
						continue
					}
					log.Println(lines[index])

					values := strings.Split(lines[index], ",")

					for field := range pluginParams {
						match := cols_regexp.FindStringSubmatch(pluginParams[field].(string))
						for k, v := range match {
							if k == 0 {
								continue
							}
							fieldIndex, _ := strconv.Atoi(v)
							//log.Println(field + ":" + v + " => " + values[fieldIndex-1])

							server.properties[field] = strings.ReplaceAll(pluginParams[field].(string), "$"+v, values[fieldIndex-1])
						}
					}

					//log.Println(server)

					log.Println("Update " + server.class + " to Neo4j")
					_, err := server.Update(neoSession, ctx)
					if err != nil {
						log.Fatal(err)
					}
				}
			case "relation":
				//pluginOutputFormat := viper.GetString("output_format")
				pluginLNode := viper.GetString("left_node")
				pluginLName := viper.GetString("left_name")
				pluginLCond := viper.GetString("left_cond")
				pluginLParams := viper.GetStringMap("left_params")
				pluginRNode := viper.GetString("right_node")
				pluginRName := viper.GetString("right_name")
				pluginRCond := viper.GetString("right_cond")
				pluginRParams := viper.GetStringMap("right_params")
				pluginRelName := viper.GetString("rel_name")
				pluginRelParams := viper.GetStringMap("rel_params")
				pluginEnableNodeCreation := viper.GetString("enable_node_creation")
				pluginEnableNodeUpdate := viper.GetString("enable_node_update")
				pluginEnableRelDelete := viper.GetString("enable_relation_delete")
				//pluginEnableRelUpdate := viper.GetString("enable_relation_update")

				cols_regexp := regexp.MustCompile(`\$(\d+)`)

				lines := strings.Split(string(out), "\n")
				for index := range lines {
					if lines[index] == "" {
						log.Println("Skip " + lines[index])
						continue
					}

					values := strings.Split(lines[index], ",")

					//log.Println(pluginLParams)
					leftNode := new(Node)
					if pluginLNode == "" {
						leftNode.class = server.class
						leftNode.name = server.name
						leftNode.cond = server.cond
						leftNode.properties = make(map[string]any)
						for k, v := range server.properties {
							leftNode.properties[k] = v
						}
					} else {
						leftNode.class = pluginLNode
						leftNode.name = pluginLName
						leftNode.cond = pluginLCond
						leftNode.properties = make(map[string]any)
						for k, v := range pluginLParams {
							leftNode.properties[k] = v
						}
					}

					match := cols_regexp.FindStringSubmatch(leftNode.name)
					for k, v := range match {
						if k == 0 {
							continue
						}
						fieldIndex, _ := strconv.Atoi(v)
						//log.Println("name: " + v + " => " + values[fieldIndex-1])

						leftNode.name = strings.ReplaceAll(leftNode.name, "$"+v, values[fieldIndex-1])
					}

					match = cols_regexp.FindStringSubmatch(pluginLCond)
					for k, v := range match {
						if k == 0 {
							continue
						}
						fieldIndex, _ := strconv.Atoi(v)
						//log.Println("cond: " + v + " => " + values[fieldIndex-1])

						leftNode.cond = strings.ReplaceAll(leftNode.cond, "$"+v, values[fieldIndex-1])
					}

					for field := range leftNode.properties {
						match := cols_regexp.FindStringSubmatch(leftNode.properties[field].(string))
						for k, v := range match {
							if k == 0 {
								continue
							}
							fieldIndex, _ := strconv.Atoi(v)
							//log.Println(field + ":" + v + " => " + values[fieldIndex-1])

							leftNode.properties[field] = strings.ReplaceAll(leftNode.properties[field].(string), "$"+v, values[fieldIndex-1])
						}
					}

					//log.Println(leftNode)

					found, err := leftNode.Exists(neoSession, ctx)
					if err != nil {
						log.Fatal(err)
					}

					if !found && pluginEnableNodeCreation == "true" {
						log.Println("Add " + leftNode.class + " to Neo4j")
						_, err := leftNode.Add(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					} else if found && pluginEnableNodeUpdate == "true" {
						log.Println("Update " + leftNode.class + " to Neo4j")
						_, err := leftNode.Update(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					}

					//log.Println(pluginRParams)
					rightNode := new(Node)
					rightNode.class = pluginRNode
					rightNode.name = pluginRName
					rightNode.cond = pluginRCond
					rightNode.properties = make(map[string]any)
					for k, v := range pluginRParams {
						rightNode.properties[k] = v
					}

					match = cols_regexp.FindStringSubmatch(rightNode.name)
					for k, v := range match {
						if k == 0 {
							continue
						}
						fieldIndex, _ := strconv.Atoi(v)
						//log.Println("name: " + v + " => " + values[fieldIndex-1])

						rightNode.name = strings.ReplaceAll(rightNode.name, "$"+v, values[fieldIndex-1])
					}

					match = cols_regexp.FindStringSubmatch(pluginRCond)
					for k, v := range match {
						if k == 0 {
							continue
						}
						fieldIndex, _ := strconv.Atoi(v)
						//log.Println("cond: " + v + " => " + values[fieldIndex-1])

						rightNode.cond = strings.ReplaceAll(rightNode.cond, "$"+v, values[fieldIndex-1])
					}

					for field := range rightNode.properties {
						match := cols_regexp.FindStringSubmatch(rightNode.properties[field].(string))
						for k, v := range match {
							if k == 0 {
								continue
							}
							fieldIndex, _ := strconv.Atoi(v)
							//log.Println(field + ":" + v + " => " + values[fieldIndex-1])

							rightNode.properties[field] = strings.ReplaceAll(rightNode.properties[field].(string), "$"+v, values[fieldIndex-1])
						}
					}

					//log.Println(rightNode)

					found, err = rightNode.Exists(neoSession, ctx)
					if err != nil {
						log.Fatal(err)
					}

					if !found && pluginEnableNodeCreation == "true" {
						log.Println("Add " + rightNode.class + " to Neo4j")
						_, err := rightNode.Add(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					} else if found && pluginEnableNodeUpdate == "true" {
						log.Println("Update " + rightNode.class + " to Neo4j")
						_, err := rightNode.Update(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					}

					currentRelationship := new(Relationship)
					currentRelationship.class = pluginRelName
					currentRelationship.left = leftNode
					currentRelationship.right = rightNode
					currentRelationship.properties = make(map[string]any)
					for k, v := range pluginRelParams {
						currentRelationship.properties[k] = v
					}

					for field := range currentRelationship.properties {
						match := cols_regexp.FindStringSubmatch(currentRelationship.properties[field].(string))
						for k, v := range match {
							if k == 0 {
								continue
							}
							fieldIndex, _ := strconv.Atoi(v)
							//log.Println(field + ":" + v + " => " + values[fieldIndex-1])

							currentRelationship.properties[field] = strings.ReplaceAll(currentRelationship.properties[field].(string), "$"+v, values[fieldIndex-1])
						}
					}

					found, err = currentRelationship.Exists(neoSession, ctx)
					if err != nil {
						log.Fatal(err)
					}

					if !found {
						if pluginEnableRelDelete == "true" {
							log.Println("Delete relation " + currentRelationship.class + " between " + currentRelationship.left.class + " " + currentRelationship.left.name + " and " + currentRelationship.right.class + " " + currentRelationship.right.name)
							_, err := currentRelationship.Delete(neoSession, ctx)
							if err != nil {
								log.Fatal(err)
							}
						} else {
							log.Println("Add relation " + currentRelationship.class + " between " + currentRelationship.left.class + " " + currentRelationship.left.name + " and " + currentRelationship.right.class + " " + currentRelationship.right.name)
							_, err := currentRelationship.Add(neoSession, ctx)
							if err != nil {
								log.Fatal(err)
							}
						}
					}

					leftNode = nil
					rightNode = nil
					currentRelationship = nil
				}
			default:
				//TO DO: move script execution out of switch case
				pluginScript := viper.GetStringMap("script")

				out, err := sshClient.executeScript(pluginScript["script"].(string))
				if err != nil {
					log.Println(err)
					continue
				}

				currentNode := new(Node)
				currentNode.class = viper.GetString("type")
				currentNode.name = viper.GetString("name")
				currentNode.cond = ""
				currentNode.properties = make(map[string]any)
				for k, v := range viper.GetStringMap("details") {
					currentNode.properties[k] = v
				}

				log.Println("Target node is " + currentNode.class)
				if pluginType != "Relation" {
					found, err := currentNode.Exists(neoSession, ctx)
					if err != nil {
						log.Fatal(err)
					}

					if !found {
						log.Println("Add node type " + pluginType)
						_, err := currentNode.Add(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					}

					currentRelationship := new(Relationship)
					currentRelationship.class = pluginScript["relation"].(string)
					currentRelationship.left = server
					currentRelationship.right = currentNode
					currentRelationship.properties = map[string]any{}

					found, err = currentRelationship.Exists(neoSession, ctx)
					if err != nil {
						log.Fatal(err)
					}

					log.Println("Script result: " + strings.TrimSuffix(out, "\n"))
					retValue := strings.TrimSuffix(out, "\n")
					if err == nil && retValue == pluginScript["truevalue"].(string) && !found {
						log.Println("Add relation between Server " + server.name + " and " + currentNode.class + " " + currentNode.name)
						_, err := currentRelationship.Add(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					} else if err == nil && retValue == pluginScript["falsevalue"].(string) && found {
						log.Println("Remove relation between Server " + server.name + " and " + currentNode.class + " " + currentNode.name)
						_, err := currentRelationship.Delete(neoSession, ctx)
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			}
		}

		server = nil
	}
}
