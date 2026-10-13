plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.example.paywall"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.example.paywall"
        minSdk = 24
        targetSdk = 35
        versionCode = 1
        versionName = "1.0"
    }
}

dependencies {
    implementation("com.revenuecat.purchases:purchases:8.10.0")
}
