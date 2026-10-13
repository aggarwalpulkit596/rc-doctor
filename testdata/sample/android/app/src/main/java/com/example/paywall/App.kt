package com.example.paywall

import android.app.Application
import com.revenuecat.purchases.Purchases
import com.revenuecat.purchases.PurchasesConfiguration

class App : Application() {
    override fun onCreate() {
        super.onCreate()
        // Copied from the iOS app by mistake: this is the App Store key.
        Purchases.configure(PurchasesConfiguration.Builder(this, "appl_AbCdEfGhIjKlMnOpQrSt").build())
    }
}
