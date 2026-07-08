package sample

// Kotlin 1.6.20+ context receivers
context(Logging)
class UserService {
    fun createUser(name: String) {
        log("Creating user: $name")
        // ... creation logic
    }
}

interface Logging {
    fun log(message: String)
}

class ConsoleLogging : Logging {
    override fun log(message: String) {
        println("[LOG] $message")
    }
}

// Multiple context receivers
context(Logging, Metrics)
fun processPayment(amount: Double) {
    log("Processing payment: $$amount")
    incrementCounter("payment.attempt")
}

interface Metrics {
    fun incrementCounter(name: String)
}

class MetricsImpl : Metrics {
    override fun incrementCounter(name: String) {
        println("[METRICS] $name: +1")
    }
}

// Usage with context declaration
context(Logging, Metrics)
fun demonstrateContext() {
    val userService = UserService()
    userService.createUser("John Doe")
    processPayment(100.0)
}
