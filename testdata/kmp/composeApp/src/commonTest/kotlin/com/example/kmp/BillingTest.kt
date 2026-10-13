package com.example.kmp

// Test-only IDs. rc-doctor must ignore this source set.
class BillingTest {
    fun lockedWithoutEntitlement() = fakeInfo().entitlements["fake_for_test"] == null
}
