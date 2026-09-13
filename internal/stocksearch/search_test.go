package stocksearch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestYahooFinderSearchAndPreview(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/v1/finance/search" {
			_, _ = writer.Write([]byte(`{"quotes":[{"symbol":"BOL.ST","longname":"Boliden AB (publ)","exchDisp":"Stockholm","currency":"SEK","quoteType":"EQUITY"}]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"chart":{"result":[{"meta":{"currency":"SEK","exchangeName":"STO","regularMarketPrice":345.6}}],"error":null}}`))
	}))
	defer server.Close()

	finder := NewYahooFinder(server.URL)
	candidates, err := finder.Search("BOL")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Symbol != "BOL.ST" {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
	preview, err := finder.Preview(candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	if preview.Price != 345.6 || preview.Currency != "SEK" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}
