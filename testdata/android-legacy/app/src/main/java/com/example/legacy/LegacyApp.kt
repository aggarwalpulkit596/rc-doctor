package com.example.legacy

import android.app.Application
import com.revenuecat.purchases.Purchases
import com.revenuecat.purchases.PurchasesAreCompletedBy
import com.revenuecat.purchases.PurchasesConfiguration

class LegacyApp : Application() {
    override fun onCreate() {
        super.onCreate()
        Purchases.configure(
            PurchasesConfiguration.Builder(this, BuildConfig.REVENUECAT_KEY)
                .appUserID("demo_user")
                .purchasesAreCompletedBy(PurchasesAreCompletedBy.MY_APP)
                .build()
        )
    }
}
