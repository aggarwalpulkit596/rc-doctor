import RevenueCat
import SwiftUI

@main
struct PaywallApp: App {
    init() {
        // A secret key copied from the dashboard instead of the public SDK key.
        Purchases.configure(withAPIKey: "sk_AbCdEfGhIjKlMnOpQrSt")
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
