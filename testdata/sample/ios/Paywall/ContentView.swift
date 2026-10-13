import RevenueCat
import SwiftUI

struct ContentView: View {
    @State private var isPremium = false

    var body: some View {
        Text(isPremium ? "Premium" : "Free")
            .task {
                let info = try? await Purchases.shared.customerInfo()
                isPremium = info?.entitlements["premium"]?.isActive == true

                // The offering was renamed in the dashboard, but not here.
                let offerings = try? await Purchases.shared.offerings()
                _ = offerings?.offering(identifier: "summer_sale")
            }
    }
}

extension ContentView {
    func restore() async {
        _ = try? await Purchases.shared.restorePurchases()
    }
}
