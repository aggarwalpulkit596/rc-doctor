package android

import (
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func setting(value, file string, line int) model.Setting {
	return model.Setting{Value: value, Location: model.Location{File: file, Line: line}}
}

func TestBuildSettingsLegacyFixture(t *testing.T) {
	facts, err := Android{}.Collect(testutil.Testdata(t, "android-legacy"))
	if err != nil {
		t.Fatal(err)
	}
	got := facts[0].Build
	want := map[string]model.Setting{
		model.BuildSDKArtifact:      setting("purchases", "app/build.gradle", 20),
		model.BuildSDKVersion:       setting("9.2.0", "app/build.gradle", 20),
		model.BuildMinSDK:           setting("21", "app/build.gradle", 12),
		model.BuildApplicationClass: setting("com.example.legacy.LegacyApp", "app/src/main/java/com/example/legacy/LegacyApp.kt", 8),
		model.BuildManifest:         setting("app/src/main/AndroidManifest.xml", "app/src/main/AndroidManifest.xml", 4),
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %+v, want %+v", k, got[k], w)
		}
	}
	for _, absent := range []string{model.BuildDesugaring, model.BuildManifestAppName} {
		if s, ok := got[absent]; ok {
			t.Errorf("%s should be absent, got %+v", absent, s)
		}
	}
	if f := facts[0]; len(f.AppUserIDs) != 1 || f.AppUserIDs[0].ID != "demo_user" ||
		!f.HasSignal(model.SignalCompletedByMyApp) || f.HasSignal(model.SignalFinishesPurchase) || !f.HasSignal(model.SignalCurrentOffering) {
		t.Errorf("facts = %+v", f)
	}
}

func TestBuildSettingsFromVersionCatalogAndKMPManifest(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"gradle/libs.versions.toml":                      "[versions]\nandroid-minSdk = \"23\"\nrevenuecat = \"8.10.0\"\n\n[libraries]\nrevenuecat = { module = \"com.revenuecat.purchases:purchases\", version.ref = \"revenuecat\" }\n",
		"composeApp/build.gradle.kts":                    "android {\n    namespace = \"com.example.shared\"\n    defaultConfig {\n        applicationId = \"com.example.app\"\n        minSdk = libs.versions.android.minSdk.get().toInt()\n    }\n    compileOptions {\n        isCoreLibraryDesugaringEnabled = true\n    }\n}\n",
		"composeApp/src/androidMain/AndroidManifest.xml": "<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\">\n    <application\n        android:name=\".MyApp\"\n        android:label=\"x\" />\n</manifest>\n",
	})
	got := BuildSettings(root, nil, nil)
	for k, w := range map[string]model.Setting{
		model.BuildMinSDK:          setting("23", "gradle/libs.versions.toml", 2),
		model.BuildSDKArtifact:     setting("purchases", "gradle/libs.versions.toml", 3),
		model.BuildSDKVersion:      setting("8.10.0", "gradle/libs.versions.toml", 3),
		model.BuildDesugaring:      setting("true", "composeApp/build.gradle.kts", 8),
		model.BuildManifestAppName: setting("com.example.shared.MyApp", "composeApp/src/androidMain/AndroidManifest.xml", 3),
	} {
		if got[k] != w {
			t.Errorf("%s = %+v, want %+v", k, got[k], w)
		}
	}
}
