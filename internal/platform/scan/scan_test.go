package scan

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

var kotlin = Patterns{
	Extensions:   []string{".kt"},
	APIKey:       []*regexp.Regexp{KeyQuoted},
	Configure:    []*regexp.Regexp{ConfigureCall},
	Entitlements: []*regexp.Regexp{EntitlementSubscript, EntitlementCall, PaywallEntitlement},
	Offerings:    []*regexp.Regexp{OfferingSubscript, OfferingCall},
}

func TestSourceAppliesSetsSkipsCommentsAndExcludes(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"app/src/main/Main.kt": `
val key = "goog_AAAAAAAAAAAAAAAAAAAA"
// val old = entitlements["commented"]
/*
val older = entitlements["blocked"]
*/
fun check() = info.entitlements["pro"]?.isActive
fun current() = info?.entitlements?.get("plus")
fun pick() = offerings["main"] ?: offerings.getOffering("backup")
Purchases.configure(config)
`,
		"app/src/test/MainTest.kt":   `val t = entitlements["test_only"]`,
		"app/build/generated/Gen.kt": `val g = entitlements["generated"]`,
		"local.properties":           "revenuecat.key=goog_BBBBBBBBBBBBBBBBBBBB\n# old.key=goog_CCCCCCCCCCCCCCCCCCCC\n",
		"notes.txt":                  `entitlements["not_code"]`,
	})
	spec := Spec{
		Exclude: regexp.MustCompile(`(^|/)test(/|$)`),
		Sets: []Patterns{
			kotlin,
			{Extensions: []string{".properties"}, APIKey: []*regexp.Regexp{KeyAssigned}},
		},
	}
	res, err := Source(root, spec)
	if err != nil {
		t.Fatal(err)
	}

	main := "app/src/main/Main.kt"
	at := func(file string, line int) model.Location { return model.Location{File: file, Line: line} }
	wantKeys := []model.APIKey{
		{Value: "goog_AAAAAAAAAAAAAAAAAAAA", Location: at(main, 2)},
		{Value: "goog_BBBBBBBBBBBBBBBBBBBB", Location: at("local.properties", 1)},
	}
	wantEnts := []model.Ref{{ID: "pro", Location: at(main, 7)}, {ID: "plus", Location: at(main, 8)}}
	wantOffs := []model.Ref{{ID: "main", Location: at(main, 9)}, {ID: "backup", Location: at(main, 9)}}
	wantConfigure := []model.Location{at(main, 10)}

	if !reflect.DeepEqual(res.APIKeys, wantKeys) {
		t.Errorf("APIKeys = %+v\nwant %+v", res.APIKeys, wantKeys)
	}
	if !reflect.DeepEqual(res.Entitlements, wantEnts) {
		t.Errorf("Entitlements = %+v\nwant %+v", res.Entitlements, wantEnts)
	}
	if !reflect.DeepEqual(res.Offerings, wantOffs) {
		t.Errorf("Offerings = %+v\nwant %+v", res.Offerings, wantOffs)
	}
	if !reflect.DeepEqual(res.ConfigureAt, wantConfigure) {
		t.Errorf("ConfigureAt = %+v, want %+v", res.ConfigureAt, wantConfigure)
	}
}

func TestSourceMissingRoot(t *testing.T) {
	if _, err := Source(t.TempDir()+"/missing", Spec{Sets: []Patterns{kotlin}}); err == nil {
		t.Fatal("expected an error for a missing root")
	}
}

func TestMatchesFirstMatchAndContains(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"a/build.gradle.kts": "// applicationId = \"com.example.commented\"\napplicationId = \"com.example.a\"\n",
		"b/build.gradle.kts": "applicationId = \"com.example.b\"\n",
		"b/other.txt":        "applicationId = \"com.example.ignored\"\n",
	})
	re := regexp.MustCompile(`applicationId\s*=\s*"([^"]+)"`)
	names := []string{"build.gradle.kts"}

	got := Matches(root, names, re)
	want := []Match{
		{Value: "com.example.a", Location: model.Location{File: "a/build.gradle.kts", Line: 2}},
		{Value: "com.example.b", Location: model.Location{File: "b/build.gradle.kts", Line: 1}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Matches = %+v\nwant %+v", got, want)
	}

	m, ok := FirstMatch(root, names, re, func(v string) bool { return v != "com.example.a" })
	if !ok || m.Value != "com.example.b" {
		t.Fatalf("FirstMatch with filter = %+v, %v", m, ok)
	}
	if _, ok := FirstMatch(root, names, re, func(string) bool { return false }); ok {
		t.Fatal("FirstMatch should fail when every match is rejected")
	}

	if !Contains(root, names, "nope", "com.example.b") {
		t.Fatal("Contains should find com.example.b")
	}
	if Contains(root, names, "com.example.ignored") {
		t.Fatal("Contains must only read the named files")
	}
}

func TestWalkStopsEarly(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{"a.kt": "", "b.kt": "", "c.kt": ""})
	var seen []string
	if err := Walk(root, nil, func(_, rel string) bool {
		seen = append(seen, rel)
		return false
	}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != "a.kt" {
		t.Fatalf("Walk visited %v, want only a.kt", seen)
	}
}

func TestPatterns(t *testing.T) {
	for _, tc := range []struct {
		name string
		re   *regexp.Regexp
		line string
		want []string
	}{
		{"kotlin string", KeyQuoted, `Builder(this, "goog_1234567890abcdefghij")`, []string{"goog_1234567890abcdefghij"}},
		{"objc string", KeyQuoted, `configureWithAPIKey:@"appl_1234567890abcdefghij"`, []string{"appl_1234567890abcdefghij"}},
		{"buildConfigField", KeyQuoted, `buildConfigField("String", "RC", "\"goog_1234567890abcdefghij\"")`, []string{"goog_1234567890abcdefghij"}},
		{"groovy single quotes", KeyQuoted, `buildConfigField 'String', 'RC', '"amzn_1234567890abcdefghij"'`, []string{"amzn_1234567890abcdefghij"}},
		{"stripe key is not ours", KeyQuoted, `"sk_live_1234567890abcdefghij"`, nil},
		{"short token", KeyQuoted, `"goog_short"`, nil},
		{"test store", KeyQuoted, `Purchases.configure(withAPIKey: "test_1234567890abcdefghij")`, []string{"test_1234567890abcdefghij"}},
		{"galaxy and mac", KeyQuoted, `"galx_1234567890abcdefghij", "mac_1234567890abcdefghij"`, []string{"galx_1234567890abcdefghij", "mac_1234567890abcdefghij"}},
		{"an ID that only looks like a key", KeyQuoted, `Question(id = "sci_PhotosynthesisLight01")`, nil},
		{"properties", KeyAssigned, `revenuecat.apiKey = goog_1234567890abcdefghij`, []string{"goog_1234567890abcdefghij"}},
		{"xcconfig", KeyAssigned, `REVENUECAT_API_KEY = appl_1234567890abcdefghij`, []string{"appl_1234567890abcdefghij"}},
		{"plist", KeyXMLText, `<string>appl_1234567890abcdefghij</string>`, []string{"appl_1234567890abcdefghij"}},
		{"string resource", KeyXMLText, `<string name="rc">goog_1234567890abcdefghij</string>`, []string{"goog_1234567890abcdefghij"}},

		{"subscript", EntitlementSubscript, `info.entitlements["pro"]?.isActive`, []string{"pro"}},
		{"active subscript", EntitlementSubscript, `info.entitlements.active["pro"] != null`, []string{"pro"}},
		{"objc subscript", EntitlementSubscript, `info.entitlements[@"pro"].isActive`, []string{"pro"}},
		{"get", EntitlementCall, `info.entitlements.get("pro")`, []string{"pro"}},
		{"safe-call get", EntitlementCall, `info?.entitlements?.get("pro")`, []string{"pro"}},
		{"containsKey", EntitlementCall, `info.entitlements.active.containsKey("pro")`, []string{"pro"}},
		{"keys.contains", EntitlementCall, `info.entitlements.active.keys.contains("pro")`, []string{"pro"}},
		{"swift paywall", PaywallEntitlement, `.presentPaywallIfNeeded(requiredEntitlementIdentifier: "pro")`, []string{"pro"}},
		{"kotlin paywall arg", PaywallEntitlement, `launcher.launchIfNeeded(requiredEntitlementIdentifier = "pro")`, []string{"pro"}},
		{"kotlin paywall builder", PaywallEntitlement, `.setRequiredEntitlementIdentifier("pro")`, []string{"pro"}},

		{"offering subscript", OfferingSubscript, `offerings["main"] ?: offerings.all["backup"]`, []string{"main", "backup"}},
		{"offering get", OfferingCall, `offerings.get("main"); offerings.getOffering("backup")`, []string{"main", "backup"}},
		{"configure", ConfigureCall, `Purchases.configure(PurchasesConfiguration.Builder(this, key).build())`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captures([]*regexp.Regexp{tc.re}, tc.line); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("captures(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
	if !ConfigureCall.MatchString(`Purchases.configure(withAPIKey: key)`) {
		t.Error("ConfigureCall should match the Swift configure call")
	}
}
