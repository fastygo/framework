package i18n

import (
	"embed"

	"github.com/fastygo/framework/pkg/web/i18n"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

type ThemeFixture struct {
	Label              string `json:"label"`
	SwitchToDarkLabel  string `json:"switch_to_dark_label"`
	SwitchToLightLabel string `json:"switch_to_light_label"`
}

type LanguageFixture struct {
	Label        string            `json:"label"`
	CurrentLabel string            `json:"current_label"`
	NextLabel    string            `json:"next_label"`
	NextLocale   string            `json:"next_locale"`
	Available    []string          `json:"available"`
	LocaleLabels map[string]string `json:"locale_labels"`
}

type ActionFixture struct {
	Label     string `json:"label"`
	Href      string `json:"href"`
	AriaLabel string `json:"aria_label"`
}

type ImageFixture struct {
	Alt string `json:"alt"`
}

type StorefrontFixture struct {
	BrandName   string          `json:"brand_name"`
	Title       string          `json:"title"`
	Theme       ThemeFixture    `json:"theme"`
	Language    LanguageFixture `json:"language"`
	Nav         []ActionFixture `json:"nav"`
	Header      HeaderFixture   `json:"header"`
	Hero        HeroFixture     `json:"hero"`
	Categories  []Category      `json:"categories"`
	Collections []Collection    `json:"collections"`
	Benefits    []Benefit       `json:"benefits"`
	Product     Product         `json:"product"`
	Rail        Rail            `json:"rail"`
	Footer      Footer          `json:"footer"`
}

type HeaderFixture struct {
	SearchLabel  string `json:"search_label"`
	AccountLabel string `json:"account_label"`
	CartLabel    string `json:"cart_label"`
	CartCount    string `json:"cart_count"`
}

type HeroFixture struct {
	Kicker       string        `json:"kicker"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Primary      ActionFixture `json:"primary"`
	SlideCurrent string        `json:"slide_current"`
	SlideTotal   string        `json:"slide_total"`
	Previous     ActionFixture `json:"previous"`
	Next         ActionFixture `json:"next"`
	Image        ImageFixture  `json:"image"`
}

type Category struct {
	Title string       `json:"title"`
	Count string       `json:"count"`
	Href  string       `json:"href"`
	Image ImageFixture `json:"image"`
}

type Collection struct {
	Label       string       `json:"label"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Href        string       `json:"href"`
	Image       ImageFixture `json:"image"`
}

type Benefit struct {
	Icon        string `json:"icon"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Product struct {
	Badge          string          `json:"badge"`
	Name           string          `json:"name"`
	Price          string          `json:"price"`
	OldPrice       string          `json:"old_price"`
	Description    string          `json:"description"`
	Gallery        []ImageFixture  `json:"gallery"`
	MainImage      ImageFixture    `json:"main_image"`
	ColorsLabel    string          `json:"colors_label"`
	Colors         []SwatchFixture `json:"colors"`
	SelectedColor  string          `json:"selected_color"`
	Quantity       Quantity        `json:"quantity"`
	AddAction      ActionFixture   `json:"add_action"`
	FavoriteAction ActionFixture   `json:"favorite_action"`
	Details        []Detail        `json:"details"`
}

type SwatchFixture struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Class    string `json:"class"`
	Selected bool   `json:"selected"`
}

type Quantity struct {
	Label      string `json:"label"`
	Value      string `json:"value"`
	MinusLabel string `json:"minus_label"`
	PlusLabel  string `json:"plus_label"`
}

type Detail struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Rail struct {
	Cart        Cart            `json:"cart"`
	Filter      Filter          `json:"filter"`
	Inspiration Inspiration     `json:"inspiration"`
	Newsletter  Newsletter      `json:"newsletter"`
	Socials     []ActionFixture `json:"socials"`
}

type Cart struct {
	Title          string        `json:"title"`
	CountLabel     string        `json:"count_label"`
	Items          []CartItem    `json:"items"`
	TotalLabel     string        `json:"total_label"`
	Total          string        `json:"total"`
	CheckoutAction ActionFixture `json:"checkout_action"`
	ViewCartAction ActionFixture `json:"view_cart_action"`
}

type CartItem struct {
	Title    string       `json:"title"`
	Subtitle string       `json:"subtitle"`
	Price    string       `json:"price"`
	Image    ImageFixture `json:"image"`
}

type Filter struct {
	Title         string          `json:"title"`
	ResetLabel    string          `json:"reset_label"`
	CategoryLabel string          `json:"category_label"`
	CategoryValue string          `json:"category_value"`
	MaterialLabel string          `json:"material_label"`
	Materials     []SwatchFixture `json:"materials"`
	Price         PriceRange      `json:"price"`
	ColorLabel    string          `json:"color_label"`
	Colors        []SwatchFixture `json:"colors"`
	ShowAllAction ActionFixture   `json:"show_all_action"`
}

type PriceRange struct {
	Label    string `json:"label"`
	MinLabel string `json:"min_label"`
	MaxLabel string `json:"max_label"`
	Value    string `json:"value"`
}

type Inspiration struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Action      ActionFixture `json:"action"`
	Image       ImageFixture  `json:"image"`
}

type Newsletter struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Placeholder string `json:"placeholder"`
	SubmitLabel string `json:"submit_label"`
}

type Footer struct {
	Copyright string        `json:"copyright"`
	Groups    []FooterGroup `json:"groups"`
}

type FooterGroup struct {
	Title string          `json:"title"`
	Links []ActionFixture `json:"links"`
}

type Bundle struct {
	Storefront StorefrontFixture
}

var Locales = []string{"en", "ru"}

var store = i18n.New[Bundle](fixtureFS, Locales, "en", func(reader i18n.Reader, loc string) (Bundle, error) {
	storefront, err := i18n.DecodeSection[StorefrontFixture](reader, loc, "storefront")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Storefront: storefront}, nil
})

func Load(locale string) (Bundle, error) {
	return store.Load(locale)
}
