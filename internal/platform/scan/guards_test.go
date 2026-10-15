package scan

import (
	"regexp"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

const testKey = "test_AbCdEfGhIjKlMnOpQrSt"

func TestKeyGuardsDebugOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
		code string // @KEY@ marks where the key goes
		want bool
	}{
		{"swift #if DEBUG", "App.swift", "#if DEBUG\nPurchases.configure(withAPIKey: \"@KEY@\")\n#else\nPurchases.configure(withAPIKey: \"appl_x\")\n#endif", true},
		{"swift #else branch", "App.swift", "#if DEBUG\nPurchases.configure(withAPIKey: \"appl_x\")\n#else\nPurchases.configure(withAPIKey: \"@KEY@\")\n#endif", false},
		{"swift #if !DEBUG", "App.swift", "#if !DEBUG\nPurchases.configure(withAPIKey: \"@KEY@\")\n#endif", false},
		{"function inside #if DEBUG", "App.swift", "#if DEBUG\nfunc configure() {\n    Purchases.configure(withAPIKey: \"@KEY@\")\n}\n#endif", true},
		{"nested #if inside #if DEBUG", "App.swift", "#if DEBUG\n#if os(iOS)\nlet x = 1\n#endif\nlet key = \"@KEY@\"\n#endif", true},
		{"#if DEBUG closed above", "App.swift", "#if DEBUG\nlet debug = true\n#endif\nlet key = \"@KEY@\"", false},
		{"unguarded", "App.kt", "class App : Application() {\n    override fun onCreate() {\n        Purchases.configure(PurchasesConfiguration.Builder(this, \"@KEY@\").build())\n    }\n}", false},
		{"BuildConfig.DEBUG block", "App.kt", "if (BuildConfig.DEBUG) {\n    key = \"@KEY@\"\n} else {\n    key = \"goog_x\"\n}", true},
		{"else branch of BuildConfig.DEBUG", "App.kt", "if (BuildConfig.DEBUG) {\n    key = \"goog_x\"\n} else {\n    key = \"@KEY@\"\n}", false},
		{"negated BuildConfig.DEBUG", "App.kt", "if (!BuildConfig.DEBUG) {\n    key = \"@KEY@\"\n}", false},
		{"same-line ternary", "App.kt", "val key = if (BuildConfig.DEBUG) \"@KEY@\" else \"goog_x\"", true},
		{"same-line, release side", "App.kt", "val key = if (BuildConfig.DEBUG) \"goog_x\" else \"@KEY@\"", false},
		{"qualified BuildConfig", "App.kt", "val key = if (com.example.BuildConfig.DEBUG) \"@KEY@\" else \"goog_x\"", true},
		{"same-line block", "App.kt", "if (BuildConfig.DEBUG) { configure(\"@KEY@\") }", true},
		{"gradle debug build type", "app/build.gradle.kts", "buildTypes {\n    debug {\n        buildConfigField(\"String\", \"RC_KEY\", \"\\\"@KEY@\\\"\")\n    }\n    release {\n        buildConfigField(\"String\", \"RC_KEY\", \"\\\"goog_x\\\"\")\n    }\n}", true},
		{"gradle release build type", "app/build.gradle.kts", "buildTypes {\n    release {\n        buildConfigField(\"String\", \"RC_KEY\", \"\\\"@KEY@\\\"\")\n    }\n}", false},
		{"gradle getByName(\"debug\")", "app/build.gradle.kts", "buildTypes {\n    getByName(\"debug\") {\n        buildConfigField(\"String\", \"RC_KEY\", \"\\\"@KEY@\\\"\")\n    }\n}", true},
		{"debug source set", "app/src/debug/java/Keys.kt", "const val KEY = \"@KEY@\"", true},
		{"flavor debug source set", "app/src/freeDebug/java/Keys.kt", "const val KEY = \"@KEY@\"", true},
		{"main source set", "app/src/main/java/Keys.kt", "const val KEY = \"@KEY@\"", false},
		{"Debug.xcconfig", "Config/Debug.xcconfig", "RC_KEY = @KEY@", true},
		{"secrets-debug.properties", "secrets-debug.properties", "RC_KEY=@KEY@", true},
		{"Release.xcconfig", "Config/Release.xcconfig", "RC_KEY = @KEY@", false},
		{"debugging.properties is not a debug config", "debugging.properties", "RC_KEY=@KEY@", false},
		{"braces in a comment don't count", "App.kt", "if (BuildConfig.DEBUG) {\n    // } else {\n    key = \"@KEY@\"\n}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := strings.ReplaceAll(tc.code, "@KEY@", testKey)
			root := testutil.WriteFiles(t, map[string]string{tc.file: code})
			line := 0
			for i, l := range strings.Split(code, "\n") {
				if strings.Contains(l, testKey) {
					line = i + 1
				}
			}
			k := model.APIKey{Value: testKey, Location: model.Location{File: tc.file, Line: line}}
			if got, _ := KeyGuards(root, k); got != tc.want {
				t.Errorf("debugOnly = %v, want %v for:\n%s", got, tc.want, code)
			}
		})
	}
}

func TestKeyGuardsMissingFile(t *testing.T) {
	k := model.APIKey{Value: testKey, Location: model.Location{File: "gone.kt", Line: 3}}
	if debug, platform := KeyGuards(t.TempDir(), k); debug || platform != "" {
		t.Error("a file that can't be read guards nothing")
	}
}

func TestKeyGuardsPlatform(t *testing.T) {
	for _, tc := range []struct {
		name, file, code, want string
	}{
		{"kotlin if on the platform", "Billing.ios.kt", "fun key() =\n  if (platform is Platform.Android) {\n    \"@KEY@\"\n  } else {\n    \"appl_x\"\n  }", "android"},
		{"the else branch is the other platform", "Billing.android.kt", "fun key() =\n  if (platform is Platform.Android) {\n    \"goog_x\"\n  } else {\n    \"@KEY@\"\n  }", "ios"},
		{"an else after an unrelated condition", "App.kt", "if (user.isPro) {\n  a()\n} else {\n  key = \"@KEY@\"\n}", ""},
		{"nested blocks inside the if branch", "App.kt", "if (isIOS) {\n  if (x) {\n    y()\n  }\n} else {\n  key = \"@KEY@\"\n}", "android"},
		{"swift #else of #if os(iOS)", "App.swift", "#if os(iOS)\nlet key = \"appl_x\"\n#else\nlet key = \"@KEY@\"\n#endif", "android"},
		{"else if names its own platform", "Billing.kt", "if (isAndroid) {\n  a()\n} else if (isIOS) {\n  key = \"@KEY@\"\n}", "ios"},
		{"swift #if os(iOS)", "App.swift", "#if os(iOS)\nlet key = \"@KEY@\"\n#endif", "ios"},
		{"innermost platform wins", "App.kt", "class AndroidApp {\n  fun f() {\n    if (Platform.isIos) {\n      k = \"@KEY@\"\n    }\n  }\n}", "ios"},
		{"no platform named", "App.kt", "class App : Application() {\n  override fun onCreate() {\n    k = \"@KEY@\"\n  }\n}", ""},
		{"the key's own line is the caller's", "App.kt", "val ANDROID_KEY = \"@KEY@\"", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := strings.ReplaceAll(tc.code, "@KEY@", testKey)
			root := testutil.WriteFiles(t, map[string]string{tc.file: code})
			line := 0
			for i, l := range strings.Split(code, "\n") {
				if strings.Contains(l, testKey) {
					line = i + 1
				}
			}
			k := model.APIKey{Value: testKey, Location: model.Location{File: tc.file, Line: line}}
			if _, got := KeyGuards(root, k); got != tc.want {
				t.Errorf("platform = %q, want %q for:\n%s", got, tc.want, code)
			}
		})
	}
}

func TestKeysReachingConfigure(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"app/build.gradle": "defaultConfig {\n  buildConfigField \"String\", \"RC_KEY\", \"\\\"test_GradleFieldKey123456\\\"\"\n}\n",
		"app/App.kt":       "Purchases.configure(PurchasesConfiguration.Builder(this, BuildConfig.RC_KEY).build())\n",
		"ios/Keys.swift":   "let debugKey = \"test_SwiftConstantKey12345\"\nlet unusedKey = \"test_UnusedConstantKey1234\"\n",
		// The constant is plain; the call that passes it is debug-only.
		"ios/App.swift": "#if DEBUG\nPurchases.configure(withAPIKey: debugKey)\n#endif\n",
	})
	set := Patterns{Extensions: []string{".kt", ".swift", ".gradle"}, APIKey: []*regexp.Regexp{KeyQuoted}, KeyIdents: KeyIdents}
	res, err := Source(root, Spec{Sets: []Patterns{set}})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]model.APIKey{}
	for _, k := range res.APIKeys {
		got[k.Value] = k
	}
	for value, want := range map[string][2]bool{ // configured, debug-only
		"test_GradleFieldKey123456":  {true, false},
		"test_SwiftConstantKey12345": {true, true},
		"test_UnusedConstantKey1234": {false, false},
	} {
		k, ok := got[value]
		if !ok || k.Configured != want[0] || k.DebugOnly != want[1] {
			t.Errorf("%s = %+v (found %v), want configured %v, debug-only %v", value, k, ok, want[0], want[1])
		}
	}
}
