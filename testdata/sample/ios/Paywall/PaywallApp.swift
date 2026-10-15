import RevenueCat
import SwiftUI

@main
struct PaywallApp: App {
    init() {
        #if DEBUG
        let apiKey = "test_AbCdEfGhIjKlMnOpQrSt" // The Test Store, in debug builds only.
        #else
        // A secret key copied from the dashboard instead of the public SDK key.
        let apiKey = "sk_AbCdEfGhIjKlMnOpQrSt"
        #endif
        Purchases.configure(withAPIKey: apiKey)
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
