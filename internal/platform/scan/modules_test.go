package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestJVMModules(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"build.gradle.kts":            "plugins { alias(libs.plugins.kotlin.jvm) apply false }\n", // the root project
		"server/build.gradle.kts":     "plugins {\n    alias(libs.plugins.kotlin.jvm)\n    alias(libs.plugins.ktor)\n    application\n}\n",
		"tools/cli/build.gradle":      "plugins {\n    id 'org.jetbrains.kotlin.jvm'\n    id 'application'\n}\n",
		"app/build.gradle.kts":        "plugins { id(\"com.android.application\") }\n",
		"shared/build.gradle.kts":     "plugins { kotlin(\"multiplatform\"); id(\"com.android.library\") }\nkotlin { jvm() }\n",
		"backend/build.gradle.kts":    "dependencies { implementation(\"io.ktor:ktor-server-core\") }\n", // says nothing: kept
		"desktopApp/build.gradle.kts": "plugins { kotlin(\"multiplatform\") }\n",
	})
	if got := JVMModules(root, nil); !reflect.DeepEqual(got, []string{"server", "tools/cli"}) {
		t.Errorf("JVMModules = %q", got)
	}
	re := ExcludeDirs(nil, []string{"server", "tools/cli"})
	for rel, want := range map[string]bool{"server/": true, "server/src/main/Main.kt": true, "tools/cli/x.kt": true, "serverless/x.kt": false, "app/server/x.kt": false} {
		if re.MatchString(rel) != want {
			t.Errorf("ExcludeDirs matches %q = %v, want %v", rel, !want, want)
		}
	}
}

func TestWalkSkipsSymlinks(t *testing.T) {
	outside := testutil.WriteFiles(t, map[string]string{"secret.properties": "KEY=goog_OutsideTheRepo1234567\n"})
	root := testutil.WriteFiles(t, map[string]string{"app/App.kt": "fun main() {}\n"})
	if err := os.Symlink(filepath.Join(outside, "secret.properties"), filepath.Join(root, "app", "linked.properties")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	var seen []string
	if err := Walk(root, nil, func(_, rel string) bool { seen = append(seen, rel); return true }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seen, []string{"app/App.kt"}) {
		t.Errorf("walked %q; a symlink can point outside the repository", seen)
	}
}
