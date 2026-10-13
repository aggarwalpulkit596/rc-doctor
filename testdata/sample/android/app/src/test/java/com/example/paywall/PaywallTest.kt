package com.example.paywall

import org.junit.Test

// Test fixtures use IDs that only exist in tests. rc-doctor must ignore this file.
class PaywallTest {
    @Test
    fun locksWithoutEntitlement() {
        val info = fakeCustomerInfo()
        assert(info.entitlements["fake_for_test"] == null)
    }
}
