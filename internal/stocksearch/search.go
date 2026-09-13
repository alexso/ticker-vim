package stocksearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Candidate struct {
	Symbol   string
	Name     string
	Exchange string
	Currency string
	Price    float64
}

type Finder interface {
	Search(query string) ([]Candidate, error)
	Preview(candidate Candidate) (Candidate, error)
}

type YahooFinder struct {
	baseURL string
	client  *http.Client
}

func NewYahooFinder(baseURL string) *YahooFinder {
	baseURL = strings.Replace(baseURL, "query1.", "query2.", 1)

	return &YahooFinder{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 8 * time.Second},
	}
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				ExchangeName       string  `json:"exchangeName"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

type yahooSearchResponse struct {
	Quotes []struct {
		Symbol              string  `json:"symbol"`
		ShortName           string  `json:"shortname"`
		LongName            string  `json:"longname"`
		ExchangeDisplayName string  `json:"exchDisp"`
		Exchange            string  `json:"exchange"`
		Currency            string  `json:"currency"`
		QuoteType           string  `json:"quoteType"`
		RegularMarketPrice  float64 `json:"regularMarketPrice"`
	} `json:"quotes"`
}

func (f *YahooFinder) Search(query string) ([]Candidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	reqURL, err := url.Parse(f.baseURL + "/v1/finance/search")
	if err != nil {
		return nil, fmt.Errorf("create Yahoo search URL: %w", err)
	}
	values := reqURL.Query()
	values.Set("q", query)
	values.Set("quotesCount", "8")
	values.Set("newsCount", "0")
	values.Set("enableFuzzyQuery", "true")
	reqURL.RawQuery = values.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Yahoo search request: %w", err)
	}
	req.Header.Set("User-Agent", "ticker-vim")

	response, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search Yahoo Finance: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo Finance search returned status %d", response.StatusCode) //nolint:goerr113
	}

	var result yahooSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Yahoo Finance search: %w", err)
	}

	candidates := make([]Candidate, 0, len(result.Quotes))
	for _, quote := range result.Quotes {
		if quote.Symbol == "" || (quote.QuoteType != "EQUITY" && quote.QuoteType != "ETF" && quote.QuoteType != "MUTUALFUND") {
			continue
		}
		name := quote.LongName
		if name == "" {
			name = quote.ShortName
		}
		exchange := quote.ExchangeDisplayName
		if exchange == "" {
			exchange = quote.Exchange
		}
		candidates = append(candidates, Candidate{
			Symbol:   quote.Symbol,
			Name:     name,
			Exchange: exchange,
			Currency: quote.Currency,
			Price:    quote.RegularMarketPrice,
		})
	}

	return candidates, nil
}

func (f *YahooFinder) Preview(candidate Candidate) (Candidate, error) {
	reqURL := f.baseURL + "/v8/finance/chart/" + url.PathEscape(candidate.Symbol) + "?range=1d&interval=1d"
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return Candidate{}, fmt.Errorf("create Yahoo preview request: %w", err)
	}
	req.Header.Set("User-Agent", "ticker-vim")
	response, err := f.client.Do(req)
	if err != nil {
		return Candidate{}, fmt.Errorf("preview Yahoo Finance quote: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Candidate{}, fmt.Errorf("yahoo Finance preview returned status %d", response.StatusCode) //nolint:goerr113
	}

	var result yahooChartResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return Candidate{}, fmt.Errorf("decode Yahoo Finance preview: %w", err)
	}
	if len(result.Chart.Result) == 0 {
		return Candidate{}, fmt.Errorf("yahoo Finance returned no quote for %s", candidate.Symbol) //nolint:goerr113
	}
	meta := result.Chart.Result[0].Meta
	candidate.Price = meta.RegularMarketPrice
	if meta.Currency != "" {
		candidate.Currency = meta.Currency
	}
	if candidate.Exchange == "" {
		candidate.Exchange = meta.ExchangeName
	}

	return candidate, nil
}
