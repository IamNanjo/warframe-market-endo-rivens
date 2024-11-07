package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	apiUrl             = "https://api.warframe.market/v1"
	rivenItemsRoute    = "/riven/items"
	rivenAuctionsRoute = "/auctions/search?"
	auctionRoute       = "https://warframe.market/auction/"
)

var (
	logger      *log.Logger = log.New(os.Stdout, "", 0)
	errorLogger *log.Logger = log.New(os.Stderr, "Error: ", 0)
)

type (
	RivenItem struct {
		Name    string `json:"item_name"`
		UrlName string `json:"url_name"`
	}
	RivenItemPayload struct {
		Items []RivenItem `json:"items"`
	}
	RivenItems struct {
		Payload RivenItemPayload `json:"payload"`
	}
)

type (
	RivenAuctionOwner struct {
		IngameName string `json:"ingame_name"`
		Status     string `json:"status"`
	}
	RivenAuctionItem struct {
		Name         string `json:"name"`
		ReRolls      int    `json:"re_rolls"`
		ModRank      int    `json:"mod_rank"`
		MasteryLevel int    `json:"mastery_level"`
	}
	RivenAuction struct {
		Id              string            `json:"id"`
		BuyoutPrice     int               `json:"buyout_price"`
		Owner           RivenAuctionOwner `json:"owner"`
		Item            RivenAuctionItem  `json:"item"`
		weapon          string            // Added later
		endoGains       int               // Calculated later
		endoPerPlatinum float64           // Calculated later
	}
	RivenAuctionPayload struct {
		Auctions []RivenAuction `json:"auctions"`
	}
	RivenAuctions struct {
		Payload RivenAuctionPayload `json:"payload"`
	}
)

type PrintAuctionParameters struct {
	auction    RivenAuction
	itemName   string
	silentMode bool
}

func logError(err error) {
	errorLogger.Printf("%q", err)
}

func doJSONRequest(apiRoute string, target interface{}) error {
	res, err := http.Get(apiUrl + apiRoute)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	json.NewDecoder(res.Body).Decode(target)

	return nil
}

func getRivenItems() (*RivenItems, error) {
	rivenItems := new(RivenItems)
	err := doJSONRequest(rivenItemsRoute, rivenItems)
	if err != nil {
		return nil, err
	}

	return rivenItems, nil
}

func getAuctions(weapon string, minReRolls string) (*RivenAuctions, error) {
	queryParamList := []string{
		"type=riven",
		"buyout_policy=direct",
		"weapon_url_name=" + weapon,
		"sort_by=price_asc",
		"re_rolls_min=" + minReRolls,
	}
	queryParams := strings.Join(queryParamList, "&")

	rivenAuctions := new(RivenAuctions)
	err := doJSONRequest(rivenAuctionsRoute+queryParams, rivenAuctions)
	if err != nil {
		return nil, err
	}

	return rivenAuctions, nil
}

func printAuction(params PrintAuctionParameters) {
	if !params.silentMode {
		logger.Println()
	}

	logger.Printf("%s%s"+
		"\n  -> Cost is %d platinum"+
		"\n  -> Amount of re-rolls is %d"+
		"\n  -> Mod rank is %d"+
		"\n  -> Endo gains %d"+
		"\n  -> Endo per platinum %.2f"+
		"\n  -> %s is %s",
		auctionRoute,
		params.auction.Id,
		params.auction.BuyoutPrice,
		params.auction.Item.ReRolls,
		params.auction.Item.ModRank,
		params.auction.endoGains,
		params.auction.endoPerPlatinum,
		params.auction.Owner.IngameName,
		params.auction.Owner.Status,
	)

	if params.auction.Owner.Status != "offline" {
		logger.Printf("  -> /w %s Hi! Are you still selling the %s %s riven for %d:platinum:?",
			params.auction.Owner.IngameName,
			params.itemName,
			params.auction.Item.Name,
			params.auction.BuyoutPrice,
		)
	}

	logger.Print("\n")
}

func clearScreen() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	cmd.Run()
}

func main() {
	startTime := time.Now()

	maxPrice := 50      // Maximum platinum price
	minReRolls := "50"  // Minimum amount of re-rolls
	silentMode := false // Only prints wanted auctions
	sortOutput := false // Sort output based on endo per plat

	positionalArg := 0
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-s":
			silentMode = true
			continue
		case "-S":
			sortOutput = true
			continue
		}

		positionalArg++

		switch positionalArg {
		case 1:
			_maxPrice, err := strconv.Atoi(arg)
			if err != nil {
				panic(err)
			}

			maxPrice = _maxPrice
		case 2:
			minReRolls = arg
		}
	}

	rivenItems, err := getRivenItems()
	if err != nil {
		panic(err)
	}

	if !silentMode {
		logger.Printf("Found %d riven items. Looking for auctions...\n", len(rivenItems.Payload.Items))
	}

	// Sleep to prevent being rate limited
	time.Sleep(time.Second / 2)

	auctionsSkipped := 0

	foundAuctions := make([]PrintAuctionParameters, 0, 5)

	for index, item := range rivenItems.Payload.Items {
		rivenAuctions, err := getAuctions(item.UrlName, minReRolls)
		if err != nil {
			panic(err)
		}

		if !silentMode && index != 0 && index%50 == 0 {
			logger.Printf("Skipped %d auctions", auctionsSkipped)
		}

		for _, auction := range rivenAuctions.Payload.Auctions {
			if auction.BuyoutPrice >= maxPrice || auction.Owner.Status == "offline" {
				auctionsSkipped++
				continue
			}

			auction.weapon = item.Name

			// math.Floor((100 × (MasteryLevel - 8) + 22.5 × 2^ModRank + 200 × ReRolls) - 7)
			auction.endoGains = int(math.Floor((100*(float64(auction.Item.MasteryLevel)-8) + 22.5*math.Pow(2, float64(auction.Item.ModRank)) + 200*float64(auction.Item.ReRolls)) - 7))
			auction.endoPerPlatinum = float64(auction.endoGains) / float64(auction.BuyoutPrice)

			foundAuction := PrintAuctionParameters{auction: auction, itemName: item.Name, silentMode: silentMode}
			foundAuctions = append(foundAuctions, foundAuction)

			printAuction(foundAuction)
		}

		// Sleep to prevent being rate limited
		time.Sleep(time.Millisecond * 350)
	}

	if sortOutput {
		clearScreen()

		sort.SliceStable(foundAuctions, func(i, j int) bool {
			return foundAuctions[i].auction.endoPerPlatinum > foundAuctions[j].auction.endoPerPlatinum
		})

		for _, auction := range foundAuctions {
			printAuction(auction)
		}
	}

	runDuration := time.Since(startTime)
	minutes := int(runDuration.Minutes())
	seconds := int(runDuration.Seconds()) % 60

	fmt.Printf("Finished after %d minutes and %d seconds\n", minutes, seconds)
	fmt.Scanln()
}
