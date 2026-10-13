import RevenueCat
import XCTest

// Test-only IDs. rc-doctor must ignore this target.
final class PaywallTests: XCTestCase {
    func testLockedWithoutEntitlement() {
        let info = makeCustomerInfo()
        XCTAssertNil(info.entitlements["fake_for_test"])
    }
}
