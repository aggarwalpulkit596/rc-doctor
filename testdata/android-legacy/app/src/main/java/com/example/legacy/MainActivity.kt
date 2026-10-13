package com.example.legacy

import androidx.activity.ComponentActivity
import com.revenuecat.purchases.CustomerInfo
import com.revenuecat.purchases.Offerings

class MainActivity : ComponentActivity() {
    fun isPro(info: CustomerInfo) = info.entitlements["pro"]?.isActive == true

    fun packagesToShow(offerings: Offerings) =
        offerings.current?.availablePackages ?: offerings["seasonal"]?.availablePackages

    fun fallback(offerings: Offerings) = offerings["default"]

    // Finishing the activity is not finishing a purchase.
    fun close() = finish()
}
