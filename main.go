package main

import (
	"encoding/json"
	"fmt"
	"log"
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
		Name    string `json:"name"`
		ReRolls int    `json:"re_rolls"`
		ModRank int    `json:"mod_rank"`
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

	maxPrice := 50
	minReRolls := "50"

	if len(os.Args) >= 2 {
		_maxPrice, err := strconv.Atoi(os.Args[1])

		if err != nil {
			panic(err)
		}

		maxPrice = _maxPrice
	}
	if len(os.Args) >= 3 {
		minReRolls = os.Args[2]
	}

	rivenItems, err := getRivenItems()
	if err != nil {
		panic(err)
	}

	// Sleep to prevent being rate limited
	time.Sleep(time.Second / 2)

	auctionsSkipped := 0

	for index, item := range rivenItems.Payload.Items {
		rivenAuctions, err := getAuctions(item.UrlName, minReRolls)
		if err != nil {
			panic(err)
		}

		if index != 0 && index%10 == 0 {
			logger.Printf("Skipped %d auctions", auctionsSkipped)
		}

		for _, auction := range rivenAuctions.Payload.Auctions {

			if auction.BuyoutPrice >= maxPrice || auction.Owner.Status == "offline" {
				auctionsSkipped++
				continue
			}

			logger.Printf(
				"\n"+auctionRoute+"%s"+
					"\n  -> %d platinum"+
					"\n  -> %d re-rolls"+
					"\n  -> %s is %s"+
					"\n  -> /w %s Hi! Are you still selling the %s %s riven for %d:platinum:?"+
					"\n\n",
				auction.Id,
				auction.BuyoutPrice,
				auction.Item.ReRolls,
				auction.Owner.IngameName,
				auction.Owner.Status,
				auction.Owner.IngameName,
				item.Name,
				auction.Item.Name,
				auction.BuyoutPrice,
			)
		}

		// Sleep to prevent being rate limited
		time.Sleep(time.Second / 3)
	}

	fmt.Printf("Finished after %s\n", time.Since(startTime))
	fmt.Scanln()
}
