package kmp

import (
	"reflect"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestDetect(t *testing.T) {
	if !(KMP{}).Detect(testutil.Testdata(t, "kmp")) {
		t.Error("the KMP fixture declares purchases-kmp")
	}
	if (KMP{}).Detect(testutil.Testdata(t, "sample")) {
		t.Error("native apps are not KMP apps")
	}
}

func TestCollectSplitsAppsAndRoutesKeys(t *testing.T) {
	facts, err := KMP{}.Collect(testutil.Testdata(t, "kmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 2 || facts[0].Platform != "android" || facts[1].Platform != "ios" {
		t.Fatalf("facts = %+v", facts)
	}
	a, i := facts[0], facts[1]
	if a.AppID != "com.example.kmp" || i.AppID != "com.example.kmp.ios" {
		t.Errorf("app ids = %q, %q", a.AppID, i.AppID)
	}

	// Both keys sit in one properties file; each goes to the app its name points at.
	keys := func(f model.Facts) (out []string) {
		for _, k := range f.APIKeys {
			out = append(out, k.Value)
		}
		return out
	}
	if got := keys(a); !reflect.DeepEqual(got, []string{"goog_KmpAndroidKey12345678"}) {
		t.Errorf("android keys = %q", got)
	}
	if got := keys(i); !reflect.DeepEqual(got, []string{"appl_KmpIosKey123456789012"}) {
		t.Errorf("ios keys = %q", got)
	}

	// commonMain uses a constant that the Gradle script defaults to "premum";
	// both apps see it, and the test source set is ignored.
	want := []model.Ref{{
		ID:       "premum",
		Location: model.Location{File: "composeApp/src/commonMain/kotlin/com/example/kmp/Billing.kt", Line: 14},
		Def: &model.Definition{
			Name:     "RC_ENTITLEMENT",
			Location: model.Location{File: "composeApp/build.gradle.kts", Line: 17},
			Text:     `"premum"`,
		},
	}}
	if !reflect.DeepEqual(a.Entitlements, want) || !reflect.DeepEqual(i.Entitlements, want) {
		t.Errorf("entitlements = %+v / %+v", a.Entitlements, i.Entitlements)
	}
	if len(a.ConfigureAt) != 1 {
		t.Errorf("ConfigureAt = %+v", a.ConfigureAt)
	}
}

func TestSharedKeyOwner(t *testing.T) {
	for _, tc := range []struct{ hint, key, want string }{
		{"ios", "goog_x123", "ios"}, // the name wins: a wrong key in the iOS slot stays the iOS app's
		{"", "appl_x123", "ios"},
		{"", "amzn_x123", "android"},
		{"", "sk_x123", ""},
	} {
		if got := owner(tc.hint, tc.key); got != tc.want {
			t.Errorf("owner(%q, %q) = %q, want %q", tc.hint, tc.key, got, tc.want)
		}
	}
}
