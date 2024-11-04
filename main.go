package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
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
		Id          string            `json:"id"`
		BuyoutPrice int               `json:"buyout_price"`
		Owner       RivenAuctionOwner `json:"owner"`
		Item        RivenAuctionItem  `json:"item"`
	}
	RivenAuctionPayload struct {
		Auctions []RivenAuction `json:"auctions"`
	}
	RivenAuctions struct {
		Payload RivenAuctionPayload `json:"payload"`
	}
)

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

func main() {
	startTime := time.Now()

	maxPrice := 50      // Maximum platinum price
	minReRolls := "50"  // Minimum amount of re-rolls
	silentMode := false // Only prints wanted auctions

	positionalArg := 0
	for _, arg := range os.Args[1:] {
		if arg == "-s" {
			silentMode = true
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

			// math.Floor((100 × (MasteryLevel - 8) + 22.5 × 2^ModRank + 200 × ReRolls) - 7)
			endoGains := int(math.Floor((100*(float64(auction.Item.MasteryLevel)-8) + 22.5*math.Pow(2, float64(auction.Item.ModRank)) + 200*float64(auction.Item.ReRolls)) - 7))
			endoPerPlatinum := float64(endoGains) / float64(auction.BuyoutPrice)

			if !silentMode {
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
				auction.Id,
				auction.BuyoutPrice,
				auction.Item.ReRolls,
				auction.Item.ModRank,
				endoGains,
				endoPerPlatinum,
				auction.Owner.IngameName,
				auction.Owner.Status,
			)

			if auction.Owner.Status != "offline" {
				logger.Printf("  -> /w %s Hi! Are you still selling the %s %s riven for %d:platinum:?",
					auction.Owner.IngameName,
					item.Name,
					auction.Item.Name,
					auction.BuyoutPrice,
				)
			}

			logger.Print("\n")
		}

		// Sleep to prevent being rate limited
		time.Sleep(time.Second / 3)
	}

	runDuration := time.Since(startTime)
	minutes := int(runDuration.Minutes())
	seconds := int(runDuration.Seconds()) % 60

	fmt.Printf("Finished after %d minutes and %d seconds\n", minutes, seconds)
	fmt.Scanln()
}
