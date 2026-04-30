package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

type LayoutData struct {
	Title          string
	Locale         string
	Active         string
	BrandName      string
	NavItems       []app.NavItem
	HeaderNavItems []app.NavItem
	ThemeToggle    view.ThemeToggleData
	LanguageToggle view.LanguageToggleData
	HeaderActions  HeaderActionsData
}

type HeaderActionsData struct {
	SearchLabel  string
	AccountLabel string
	CartLabel    string
	CartCount    string
}

type Page struct {
	Hero        Hero
	Categories  []Category
	Collections []Collection
	Benefits    []Benefit
	Product     Product
	Rail        Rail
	Footer      Footer
}

type Action struct {
	Label     string
	Href      string
	AriaLabel string
}

type Image struct {
	Alt string
}

type Hero struct {
	Kicker       string
	Title        string
	Description  string
	Primary      Action
	SlideCurrent string
	SlideTotal   string
	Previous     Action
	Next         Action
	Image        Image
}

type Category struct {
	Title string
	Count string
	Href  string
	Image Image
}

type Collection struct {
	Label       string
	Title       string
	Description string
	Href        string
	Image       Image
}

type Benefit struct {
	Icon        string
	Title       string
	Description string
}

type Product struct {
	Badge          string
	Name           string
	Price          string
	OldPrice       string
	Description    string
	Gallery        []Image
	MainImage      Image
	ColorsLabel    string
	Colors         []Swatch
	SelectedColor  string
	Quantity       Quantity
	AddAction      Action
	FavoriteAction Action
	Details        []Detail
}

type Swatch struct {
	Label    string
	Value    string
	Class    string
	Selected bool
}

type Quantity struct {
	Label      string
	Value      string
	MinusLabel string
	PlusLabel  string
}

type Detail struct {
	Label string
	Value string
}

type Rail struct {
	Cart        Cart
	Filter      Filter
	Inspiration Inspiration
	Newsletter  Newsletter
	Socials     []Action
}

type Cart struct {
	Title          string
	CountLabel     string
	Items          []CartItem
	TotalLabel     string
	Total          string
	CheckoutAction Action
	ViewCartAction Action
}

type CartItem struct {
	Title    string
	Subtitle string
	Price    string
	Image    Image
}

type Filter struct {
	Title         string
	ResetLabel    string
	CategoryLabel string
	CategoryValue string
	MaterialLabel string
	Materials     []Swatch
	Price         PriceRange
	ColorLabel    string
	Colors        []Swatch
	ShowAllAction Action
}

type PriceRange struct {
	Label    string
	MinLabel string
	MaxLabel string
	Value    string
}

type Inspiration struct {
	Title       string
	Description string
	Action      Action
	Image       Image
}

type Newsletter struct {
	Title       string
	Description string
	Placeholder string
	SubmitLabel string
}

type Footer struct {
	BrandName string
	Copyright string
	Groups    []FooterGroup
}

type FooterGroup struct {
	Title string
	Links []Action
}
