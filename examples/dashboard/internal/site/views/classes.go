package views

import dashboardblocks "github.com/fastygo/blocks/dashboard"

func dashboardPageClasses() dashboardblocks.PageClasses {
	return dashboardblocks.PageClasses{
		Page:        "dashboard-page",
		Header:      "dashboard-page-header",
		Title:       "dashboard-page-title",
		Description: "dashboard-page-description",
	}
}
