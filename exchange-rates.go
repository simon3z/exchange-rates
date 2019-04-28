package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type CurrencyDate struct {
	time.Time
	RawValue string
}

func (d *CurrencyDate) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &d.RawValue); err != nil {
		return err
	}

	t, err := time.Parse("2006-01-02", d.RawValue)

	if err != nil {
		return err
	}

	d.Time = t

	return nil
}

type CurrencyRate struct {
	IsoCode       string       `json:"isoCode"`
	AvgRate       string       `json:"avgRate"`
	ReferenceDate CurrencyDate `json:"referenceDate"`
}

type DailyRates struct {
	Rates []CurrencyRate `json:rates`
}

func readDateLine(r *bufio.Reader) (*time.Time, error) {
	text, _, err := r.ReadLine()

	if err != nil {
		return nil, err
	}

	t, err := time.Parse("02/01/2006", strings.TrimSpace(string(text)))

	if err != nil {
		return nil, err
	}

	return &t, nil
}

func GetDailyRates(client *http.Client, date *time.Time, currency, baseCurrency string) (*DailyRates, error) {
	values := url.Values{
		"referenceDate":   []string{date.Format("2006-01-02")},
		"currencyIsoCode": []string{currency},
	}

	if baseCurrency != "" {
		values.Add("baseCurrencyIsoCode", baseCurrency)
	}

	query := url.URL{
		Scheme:   "https",
		Host:     "tassidicambio.bancaditalia.it",
		Path:     "/terzevalute-wf-web/rest/v1.0/dailyRates",
		RawQuery: values.Encode(),
	}

	req, err := http.NewRequest("GET", query.String(), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	if res.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("error: %s", strings.TrimSpace(string(body)))
	}

	rates := new(DailyRates)
	err = json.Unmarshal(body, rates)

	if err != nil {
		panic(err)
	}

	return rates, nil
}

var CmdFlags = struct {
	BaseCurrency string
	Currency     string
}{}

func init() {
	flag.StringVar(&CmdFlags.Currency, "c", "", "Currency ISO Code (e.g. EUR)")
	flag.StringVar(&CmdFlags.BaseCurrency, "b", "", "Base Currency ISO Code (optional e.g. USD)")
}

func main() {
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	client := new(http.Client)

	for {
		t, err := readDateLine(reader)

		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		rates, err := GetDailyRates(client, t, CmdFlags.Currency, CmdFlags.BaseCurrency)

		if err != nil {
			panic(err)
		}

		multipleCurrencies := len(rates.Rates) > 1

		for _, i := range rates.Rates {
			if i.ReferenceDate.Time != *t {
				panic(fmt.Errorf("unexpected rate date: %s", t))
			}

			if multipleCurrencies {
				println(i.IsoCode, i.AvgRate)
			} else {
				println(i.AvgRate)
			}
		}
	}
}
