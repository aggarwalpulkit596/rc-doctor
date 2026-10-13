plugins {
    kotlin("multiplatform")
    id("com.android.application")
}

android {
    namespace = "com.example.kmp"
    defaultConfig {
        applicationId = "com.example.kmp"
    }
}

// Generates BillingConfig.kt from secrets.properties (not committed).
val generateBillingConfig by tasks.registering {
    val props = java.util.Properties()
    rootProject.file("secrets.properties").takeIf { it.exists() }?.inputStream()?.use { props.load(it) }
    val entitlement = props.getProperty("RC_ENTITLEMENT", "premum")
}
