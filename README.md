<p>
    <a href="https://github.com/alexso/ticker-vim/releases"><img src="https://img.shields.io/github/v/release/alexso/ticker-vim" alt="Latest Release"></a>
    <a href="https://github.com/alexso/ticker-vim/actions"><img src="https://github.com/alexso/ticker-vim/actions/workflows/test.yml/badge.svg" alt="Build Status"></a>
</p>

<h1 align="center">ticker-vim</h1>
<p align="center">
Ticker with Vim navigation, fuzzy filtering, and visible watchlist groups
</p>
<p align="center">
<img align="center" src="./docs/ticker.gif" />
</p>

## What is ticker-vim?

`ticker-vim` is a fork of [Ticker](https://github.com/achannarasappa/ticker),
focused on keyboard-driven watchlist management. It keeps Ticker's market data,
portfolio, group, and configuration features while adding Vim navigation, fuzzy
filtering, visible group tabs, and in-app editing.

Ticker was created by Achanna Rasappa and contributors. This fork remains
licensed under GPL-3.0.

## Differences from Ticker

|Feature|ticker-vim|
|---|---|
|Navigation|Select rows with `j`/`k`, jump with `u`/`d`, and go to the first or last row with `g`/`G`|
|Selection|Highlights the complete selected row with a configurable background color|
|Groups|Shows groups as numbered footer tabs; switch with `h`/`l`, `Tab`/`Shift-Tab`, or `1`–`9`|
|Filtering|Fuzzy-filter the current group by symbol or display name with `/`|
|Watchlist editing|Search, preview, and add with `a`; delete with `x`; edit a display name with `e`|
|Names|Uses inline `.ticker.yaml` comments as editable display names|
|Display order|Toggle name-first or symbol-first rows with `t`, and remember the choice|
|Sorting|Starts alphabetically and remembers the chosen sorting separately for each group|
|Quote time|Can show Yahoo's timestamp for each quote, including its date when it is not from today|
|Configuration|Stores fork-specific preferences separately in `ticker-vim.yaml`|

The original live quotes, pre/post-market data, positions, multiple cost-basis
lots, summaries, groups, currency conversion, and color schemes remain available.

## Install

Install with Homebrew:

```sh
brew install alexso/tap/ticker-vim
```

Apple Silicon and Intel macOS archives are also available from the
[releases page](https://github.com/alexso/ticker-vim/releases).

`ticker-vim` remains compatible with Ticker's `.ticker.yaml`. It searches the
home directory, current directory, `$XDG_CONFIG_HOME`, and
`$XDG_CONFIG_HOME/ticker`, so this location works without an extra flag:

```text
~/.config/ticker/.ticker.yaml
```

## Quick Start

```sh
ticker-vim -w NET,AAPL,TSLA
```

## Keyboard shortcuts

|Key|Action|
|---|---|
|`j` / `k`|Select the next / previous stock|
|`u` / `d`|Jump up / down half a visible screen|
|`g` / `G`|Jump to top / bottom|
|`h` / `l`|Previous / next group|
|`Tab` / `Shift-Tab`|Next / previous group|
|`1`–`9`|Open that numbered group directly|
|`/`|Start fuzzy filtering|
|`a`|Search, preview, and add a stock to the active group|
|`x`|Preview and delete the selected stock from the active group|
|`e`|Edit the selected stock's display name|
|`t`|Toggle whether the name or symbol appears first|
|`s`|Cycle the active group's sorting|
|`Enter`|Accept the current filter or dialog choice|
|`Esc`|Clear the filter or close a dialog|
|`q`|Quit|

All fork-added shortcuts can be remapped in `ticker-vim.yaml`. The original
`Tab`, `Shift-Tab`, `s`, `q`, `Enter`, and `Esc` controls are not remapped there.

## ticker-vim configuration

Fork-specific preferences live in `ticker-vim.yaml`, beside the `.ticker.yaml`
that was selected at startup. For the XDG path above, the files are:

```text
~/.config/ticker/.ticker.yaml
~/.config/ticker/ticker-vim.yaml
```

|Section|Controls|Saved automatically?|
|---|---|---|
|`keybindings`|Remappable shortcuts added by ticker-vim|No|
|`highlight`|Selected-row background color|No|
|`sorting`|Default sorting and per-group overrides|Per-group overrides are saved when you press `s`|
|`display`|Name/symbol order and quote timestamps|`first-line` is saved when you press `t`|

Every setting is optional. This example contains all defaults:

```yaml
keybindings:
  select-up: k             # Select the previous stock
  select-down: j           # Select the next stock
  select-first: g          # Jump to the first stock
  select-last: G           # Jump to the last stock
  page-up: u               # Jump up half a visible screen
  page-down: d             # Jump down half a visible screen
  previous-group: h        # Open the previous group
  next-group: l            # Open the next group
  groups: ["1", "2", "3", "4", "5", "6", "7", "8", "9"] # Direct shortcuts for groups 1–9
  filter: "/"              # Open fuzzy filtering
  add-stock: a             # Search for and add a stock
  delete-stock: x          # Delete the selected stock from this group
  edit-name: e             # Edit the selected stock's display name
  toggle-first-line: t     # Toggle between name-first and symbol-first rows

highlight:
  background: "#142350"    # Selected-row color; use a #RRGGBB hex value

sorting:
  default: alpha           # alpha, change, value, or user
  groups: {}               # Per-group overrides are added when you press s

display:
  first-line: name         # name or symbol
  quote-time: true         # true or false; requires show-tags in .ticker.yaml
```

Key names use Bubble Tea notation, for example `ctrl+n`, `alt+1`, `shift+tab`,
and `enter`. Group shortcuts map by position, so the first entry opens the first
group, the second opens the second group, and so on.

### Row display and highlighting

`highlight.background` sets the selected row's background using a `#RRGGBB`
color. `display.first-line` accepts `name` or `symbol`; pressing the configured
toggle key saves the new value immediately.

When `display.quote-time` is `true` and `show-tags: true` is enabled in
`.ticker.yaml`, the middle tag contains the quote timestamp instead of `Live`.
Today's quotes show only the time. Older quotes also show the date, and exchanges
with delayed data include the delay beside the timestamp. If the data source does
not provide a timestamp, the tag falls back to `Live` or `Delayed`.

### Display names

An inline watchlist comment is used as that symbol's display name and takes
precedence over Yahoo's name:

```yaml
watchlist:
  - AAPL # Apple Inc.
```

Pressing `e` edits this comment. This is valid YAML and remains compatible with
Ticker, which simply treats it as a comment.

### Adding and deleting stocks

The add-stock dialog accepts a public symbol or company name, searches Yahoo
Finance, and displays canonical exchange-specific symbols such as `BOL.ST`. It
shows a current-price preview and asks for confirmation before updating the active
group in `.ticker.yaml`. New entries include Yahoo's company or fund name as an
inline YAML comment.

Press `x` to preview and confirm removal of the highlighted stock from only the
active group's watchlist in `.ticker.yaml`; the same symbol in other groups is
untouched.

### Sorting

|Value|Order|
|---|---|
|`alpha`|Alphabetical by whichever value is displayed first: name or symbol|
|`change`|Daily percentage change, with closed markets last|
|`value`|Position value, highest first|
|`user`|The order written in `.ticker.yaml`|

`sorting.default` applies to every group that has no override. Pressing `s`
cycles the active group's sorting and saves its choice under `sorting.groups`.
For example:

```yaml
sorting:
  default: alpha
  groups:
    Stockholm: alpha
    USA: value
    funds: change
```

In this example, any group other than Stockholm, USA, and funds uses `alpha`.
The `groups` map starts as `{}` and is populated automatically as sorting choices
are changed. Sorting only changes `ticker-vim.yaml`; it does not reorder or
otherwise modify `.ticker.yaml`. These UI preferences take precedence over the
older `sort:` setting in `.ticker.yaml`.

## Stock and portfolio configuration

Market data, groups, positions, currency conversion, tags, and original color
settings remain in `.ticker.yaml`. The interface preferences above intentionally
use a separate file, making it easy to share the stock configuration with the
original Ticker.

## Usage
|Option Name|Alias|Flag|Default|Description|
|-------------------|--|-------------------|----------------|-------------------------------------------------|
|                   |  |--config           |`~/.ticker.yaml`|config file location with watchlist and positions|
|`interval`         |-i|--interval         |`5`             |Refresh interval in seconds|
|`watchlist`        |-w|--watchlist        |                |comma separated list of symbols to watch|
|`show-tags`        |  |--show-tags        |                |display currency, quote timestamp or delay, and exchange name for each quote |
|`show-fundamentals`|  |--show-fundamentals|                |display open price, previous close, and day range |
|`show-separator`   |  |--show-separator   |                |layout with separators between each quote|
|`show-summary`     |  |--show-summary     |                |show total day change, total value, and total value change|
|`show-positions`   |  |--show-positions   |                |show positions including weight, average cost, and quantity|
|`sort`             |  |--sort             |                |sort quotes on the UI - options are change percent (default), `alpha`, `value`, and `user`|
|`version`          |  |--version          |                |print the current version number|
|`cache`            |  |--no-cache         |`true`          |cache data retrieved at startup|
|`debug`            |  |--debug            |                |enable debug logging to `./ticker-log-<date>.log`|

## Configuration

Configuration is not required to watch stock price but is helpful when always watching the same stocks. Configuration can also be used to set cost basis lots which will in turn be used to show total gain or loss on any position.

```yaml
# ~/.ticker.yaml
show-summary: true
show-tags: true
show-fundamentals: true
show-separator: true
show-positions: true
interval: 5
currency: USD
currency-summary-only: false
watchlist:
  - NET
  - TEAM
  - ESTC
  - BTC-USD # Bitcoin price via Yahoo
  - SOL.X # Solana price via Coinbase
  - BIT-30MAY25-CDE.CB # Bitcoin futures contract price via Coinbase
lots:
  - symbol: "ABNB"
    quantity: 35.0
    unit_cost: 146.00
  - symbol: "ARKW"
    quantity: 20.0
    unit_cost: 152.25
  - symbol: "ARKW"
    quantity: 20.0
    unit_cost: 145.35
    fixed_cost: 7.00 # e.g. brokerage commission fee
groups:
  - name: crypto
    watchlist:
      - SHIB-USD
      - VGX-USD
    lots:
      - symbol: SOL1-USD
        quantity: 17
        unit_cost: 159.10
```

* All properties in `.ticker.yaml` are optional
* Symbols not on the watchlist that exists in `lots` are implicitly added to the watchlist
* To add multiple cost basis lots (`quantity`, `unit_cost`) for the same `symbol`, include two or more entries - see `ARKW` example above
* `.ticker.yaml` can be set in user home directory, the current directory, or [XDG config home](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)
* Quantities can be negative to represent closed positions (position netting), short positions, borrowed assets, and other concepts

### Expanded display

With  `--show-summary`, `--show-tags`, `--show-fundamentals`, `--show-positions`, and `--show-separator` options set, the layout and information displayed expands:

<img src="./docs/ticker-all-options.png" />

### Sorting

It's possible to set a custom sort order with the `--sort` flag or `sort:` config option with these options:

* Default - change percent with closed markets at the end
* `alpha` to sort alphabetically by symbol
* `value` to sort by position value
* `user` to sort by the order defined in configuration with positions on first then watched symbols

### Groups

Watchlists and lots can be grouped in `.ticker.yaml` under the `groups` property. While running `ticker-vim`, press <kbd>TAB</kbd> to cycle forward through groups or <kbd>SHIFT+TAB</kbd> to cycle backward, or use the group shortcuts described above.

* If top level `watchlist` or `lots` properties are defined in the configuration file, the entries there will be added to a group named `default` which will always be shown first
* Ordering is defined by order in the configuration file

### Data Sources & Symbols

`ticker-vim` pulls market data from a few different sources with Yahoo Finance as the default. Symbols for non-default data sources follow the format `<symbol>.<source>` where `<symbol>` is the canonical symbol within that data source and `<source>` is the data source specifier. Below is a list of the supported data sources and their specifiers:

* *none* - symbols with no suffix will default to Yahoo Finance as the data source
* `.X` - symbols with this suffix are shorthand symbols inherited from Ticker and intended to provide more concise and familiar symbols for popular assets (e.g. using `SOL.X` rather than `SOLANA.CG`)
  * The full list of ticker symbols can be found [here](https://github.com/achannarasappa/ticker-static/blob/master/symbols.csv). Initial values are populated with the top cryptocurrencies by volume on Coinbase at the time of update
* `.CB` - symbols with this suffix will use Coinbase as the data source. The symbol can be found by searching for the asset on [Coinbase](https://www.coinbase.com/explore/s/listed) and finding the symbol for the asset. (e.g. for Starknet check the [market page](https://www.coinbase.com/advanced-trade/spot/STRK-USD) to find the symbol `STRK` and set the symbol to `STRK.CB` in ticker-vim).

Note: Coincap (`.CC`) and CoinGecko (`.CG`) are no longer supported after v5.0.0

### Currency Conversion

`ticker-vim` supports converting from the exchange's currency to a local currency. This can be set by setting the `currency` property in `.ticker.yaml` to an [ISO 4217 3-digit currency code](https://docs.1010data.com/1010dataReferenceManual/DataTypesAndFormats/currencyUnitCodes.html).

<img src="./docs/ticker-currency.png" />

* When a `currency` is defined, all values are converted including summary, quote, and position
* Add cost basis lots in the currency of the exchange - these will be converted automatically when `currency` is defined
* If a `currency` is not set (default behavior) and the `show-summary` option is enabled, the summary will be calculated in USD regardless of the exchange currency to avoid mixing currencies
* Currencies are retrieved only once at start time - currency exchange rates do fluctuate over time and thus converted values may vary depending on when ticker is started
* If the `currency-summary-only` is set to `true` and a value is set for `currency`, only the summary values will be converted
* If `currency-disable-unit-cost-conversion` flag to `true`, currency conversion will not be done when calculating the cost basis. This can be useful for users that purchase a non-US asset and want to use the currency exchange rate at the time of purchase by inputting the unit cost in their local currency (set in `currency`) rather than using the most recent currency exchange rate.

#### Minor currencies

`ticker-vim` supports quotes returned in a currency's minor unit rather than its major unit (e.g. London Stock Exchange listings quoted in pence as `GBp` instead of pounds as `GBP`). Minor unit conversion to major unit behavior can be controlled similarly to the currency conversion feature.

These are the behaviors for a minor unit quote:
* No `currency` set (or `currency-summary-only: true`) - per-position values, quote price, and the cost basis are all displayed in the minor unit and cost basis should be set in the minor unit
* `currency` is set - all displayed values are converted to the major unit. Cost basis should still be entered in the minor unit
* `currency` is set with `currency-disable-unit-cost-conversion` - displayed values are converted to the major unit and cost basis should be set in the major unit

### Cache

`ticker-vim` caches reference data to a single local file shared between sessions to speed up startup. The cache can be disabled with the `--no-cache` flag or `cache: false` in `.ticker.yaml`.

### Custom Color Schemes

`ticker-vim` supports setting custom color schemes from the stock config file. Colors are represented by a [hex triplet](https://en.wikipedia.org/wiki/Web_colors#Hex_triplet). Below is an annotated example config block from `.ticker.yaml` where custom colors are set:

```yaml
# ~/.ticker.yaml
watchlist:
  - NET
  - TEAM
  - ESTC
  - BTC-USD
colors:
  text: "#005fff"
  text-light: "#0087ff"
  text-label: "#00d7ff"
  text-line: "#00ffff"
  text-tag: "#005fff"
  background-tag: "#0087ff"
```

* Terminals supporting TrueColor will be able to represent the full color space and in other cases colors will be down sampled
* Any omitted or invalid colors will revert to default color scheme values

### Printing Positions

`ticker-vim` supports printing positions to the terminal as text with `ticker-vim print`. Output defaults to JSON but CSV output can also be generated by passing the `--format=csv` flag.

```sh
$ ticker-vim --config=./.ticker.yaml print
[{"name":"Airbnb, Inc.","symbol":"ABNB","price":164.71,"value":16965.13,"cost":15038,"quantity":103,"weight":53.66651978212161},{"name":"Tesla, Inc.","symbol":"TSLA","price":732.35,"value":14647,"cost":15660,"quantity":20,"weight":46.33348021787839}]
```

* Ensure there is at least one lot in the configuration file in order to generate output
* A specific config file can be specified with the `--config` flag

## Notes

* **Market data delay**
  * _Yahoo Finance_ - Market data pulled from Yahoo Finance will have some lag (<~30s) introduced by intermediary systems and certain exchanges will impose intentional delays on data. NYSE and NASDAQ offer real-time market data but other exchanges may not. Consult the [help article](https://help.yahoo.com/kb/SLN2310.html) on exchange delays to determine which exchanges you can expect delays for or use the `--show-tags` flag to include each quote's timestamp and reported delay. Yahoo Finance also relies on polling, so `interval` determines the polling frequency.
  * _Coinbase_ - Market data for spot assets on Coinbase is directly streamed from the exchange through a WebSocket connection and is available in near real-time. Derivatives assets (i.e. symbols with `-CDE` suffix) are polling based however Basis is updated in near real-time based on spot market data changes
* **Non-US Symbols, Forex, ETFs** - Their Yahoo names and symbols may differ from the familiar public names. Use `a` to search or check [Yahoo Finance](https://finance.yahoo.com/) for the canonical symbol.
* **Terminal fonts** - Font with support for the [`HORIZONTAL LINE SEPARATOR` unicode character](https://www.fileformat.info/info/unicode/char/23af/fontsupport.htm) is required to properly render separators (`--show-separator` option)

## Integrations

* [alpaca-ticker-config](https://www.npmjs.com/package/alpaca-ticker-config) - Pull [alpaca.markets](https://alpaca.markets) positions into `.ticker.yaml` from the command line

## Updating from upstream

The upstream project is [achannarasappa/ticker](https://github.com/achannarasappa/ticker).
To incorporate a newer upstream version, fetch its tags and merge the desired tag
or branch into a separate update branch. Resolve any conflicts there, then run the
full tests and lint checks before merging it into the fork's main development
branch. Keeping fork-specific UI settings in `ticker-vim.yaml` reduces overlap
with upstream's `.ticker.yaml` format.

## Development

Running tests:
```sh
go tool ginkgo -cover ./...
```

Linting:
```sh
go tool golangci-lint run
```

## Libraries ticker-vim uses

* [bubbletea](https://github.com/charmbracelet/bubbletea) - terminal UI framework
* [termenv](https://github.com/muesli/termenv) - color and styling for the terminal
* [term-grid](https://github.com/achannarasappa/term-grid) - grid layout library terminal UIs

## Related Tools

* [tickrs](https://github.com/tarkah/tickrs) - real-time terminal stock ticker with support for graphing, options, and other analysis information
* [cointop](https://github.com/miguelmota/cointop) - terminal UI tracking cryptocurrencies
