package main

import (
	"bufio"
	"fmt"
	"os"
	"path"

	"github.com/pilosa/go-pilosa"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run startrace.go PATH_TO_DATASET [PILOSA_ADDRESS]")
		os.Exit(1)
	}
	datasetPath := os.Args[1]
	address := ":10101"
	if len(os.Args) > 2 {
		address = os.Args[2]
	}
	uri, err := pilosa.NewURIFromAddress(address)
	checkErr(err)
	client := pilosa.NewClientWithURI(uri)
	languageNames, err := loadLanguageNames(datasetPath)
	checkErr(err)
	runQueries(client, languageNames)
}

func loadLanguageNames(datasetPath string) ([]string, error) {
	languageNames := make([]string, 0)
	path := path.Join(datasetPath, "languages.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		languageNames = append(languageNames, scanner.Text())
	}
	return languageNames, nil
}

func runQueries(client *pilosa.Client, languageNames []string) {
	var err error
	var response *pilosa.QueryResponse
	var query pilosa.PQLQuery
	var repositoryIDs []uint64

	// Let's load the schema from the server.
	schema, err := client.Schema()
	if err != nil {
		// Most calls will return an error value.
		// You should handle them appropriately.
		// We will just terminate the program in this case.
		panic(err)
	}

	// We need to refer to indexes and frames before we can use them in a query.
	repository, _ := schema.Index("repository", nil)
	stargazer, _ := repository.Frame("stargazer", nil)
	language, _ := repository.Frame("language", nil)

	// Which repositories did user 14 star:
	response, err = client.Query(stargazer.Bitmap(14), nil)
	checkErr(err)
	repositoryIDs = response.Result().Bitmap.Bits
	fmt.Println("User 14 starred:")
	printIDs(repositoryIDs)

	fmt.Println()

	// What are the top 5 languages in the sample data:
	response, err = client.Query(language.TopN(5), nil)
	languages := response.Result().CountItems
	fmt.Println("Top Languages:")
	printTopNLanguages(languages, languageNames)

	fmt.Println()

	// Which repositories were starred by both user 14 and 19:
	query = repository.Intersect(
		stargazer.Bitmap(14),
		stargazer.Bitmap(19),
	)
	response, err = client.Query(query, nil)
	repositoryIDs = response.Result().Bitmap.Bits
	fmt.Println("Both user 14 and 19 starred:")
	printIDs(repositoryIDs)

	fmt.Println()

	// Which repositories were starred by either user 14 or 19:
	query = repository.Union(
		stargazer.Bitmap(14),
		stargazer.Bitmap(19),
	)
	response, err = client.Query(query, nil)
	repositoryIDs = response.Result().Bitmap.Bits
	fmt.Println("User 14 or 19 starred:")
	printIDs(repositoryIDs)

	fmt.Println()

	// Which repositories were starred by user 14 or 19 and were written in language 1:
	query = repository.Intersect(
		repository.Union(
			stargazer.Bitmap(14),
			stargazer.Bitmap(19),
		),
		language.Bitmap(1),
	)
	response, err = client.Query(query, nil)
	repositoryIDs = response.Result().Bitmap.Bits
	fmt.Println("User 14 or 19 starred, written in language 1:")
	printIDs(repositoryIDs)

	fmt.Println()

	// Set user 99999 as a stargazer for repository 77777:
	_, err = client.Query(stargazer.SetBit(99999, 77777), nil)
	checkErr(err)
	fmt.Printf("Set user 99999 as a stargazer for repository 77777\n\n")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func printIDs(ids []uint64) {
	for i, id := range ids {
		fmt.Printf("\t%d. %d\n", i+1, id)
	}
}

func printTopNLanguages(items []*pilosa.CountResultItem, languageNames []string) {
	for i, item := range items {
		fmt.Printf("\t%d. %s (%d stars)\n", i+1, languageNames[item.ID], item.Count)
	}
}
