package com.example.kmp

import com.revenuecat.purchases.kmp.Purchases
import com.revenuecat.purchases.kmp.PurchasesConfiguration
import com.revenuecat.purchases.kmp.models.CustomerInfo

expect fun revenueCatKey(): String

fun configureBilling() {
    Purchases.configure(PurchasesConfiguration.Builder(apiKey = revenueCatKey()).build())
}

fun isPremium(info: CustomerInfo): Boolean =
    info.entitlements[BillingConfig.RC_ENTITLEMENT]?.isActive == true

suspend fun restore() = Purchases.sharedInstance.awaitRestore()
