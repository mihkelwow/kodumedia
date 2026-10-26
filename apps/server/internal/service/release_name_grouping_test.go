package service

import "testing"

func TestCleanSeriesTitleKeepsDotSeparatedFinalTitleWord(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw   string
		title string
		year  int
	}{
		{raw: "IT.Welcome.to.Derry", title: "IT Welcome to Derry"},
		{raw: "For.All.Mankind", title: "For All Mankind"},
		{raw: "Doctor.Who.2005", title: "Doctor Who", year: 2005},
	}
	for _, test := range cases {
		test := test
		t.Run(test.raw, func(t *testing.T) {
			t.Parallel()
			title, year := cleanSeriesTitle(test.raw)
			if title != test.title || year != test.year {
				t.Fatalf("cleanSeriesTitle(%q) = %q, %d; want %q, %d", test.raw, title, year, test.title, test.year)
			}
		})
	}
}

func TestReleaseWebsitePrefixDoesNotConsumeTitleWordTo(t *testing.T) {
	t.Parallel()
	candidate, _, ok := parseMetadataCandidate("IT.Welcome.to.Derry", 0)
	if !ok || candidate.Title != "IT Welcome to Derry" {
		t.Fatalf("title was mistaken for a .to website prefix: %+v ok=%v", candidate, ok)
	}

	websiteCandidate, _, ok := parseMetadataCandidate("www.usabit.com - Lotte.And.The.Moonstone.Secret.2011.mkv", 0)
	if !ok || websiteCandidate.Title != "Lotte And The Moonstone Secret" || websiteCandidate.Year != 2011 {
		t.Fatalf("real release website prefix was not removed: %+v ok=%v", websiteCandidate, ok)
	}
}
