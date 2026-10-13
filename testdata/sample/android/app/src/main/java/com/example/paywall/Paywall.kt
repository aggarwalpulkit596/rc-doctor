package com.example.paywall

import com.revenuecat.purchases.CustomerInfo
import com.revenuecat.purchases.Offering
import com.revenuecat.purchases.Offerings

/* Old check, kept for reference:
fun wasPro(info: CustomerInfo) = info.entitlements["pro_legacy"]?.isActive == true
*/

fun isPremium(info: CustomerInfo): Boolean =
    info.entitlements["premum"]?.isActive == true

fun paywallOffering(offerings: Offerings): Offering? =
    offerings["default"] ?: offerings.current
