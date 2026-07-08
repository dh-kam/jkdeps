package sample

// Kotlin 1.5+ value class (inline class)
@JvmInline
value class UserId(val id: Long)

// Kotlin 1.5+ sealed interface
sealed interface Result

data class Success(val value: String) : Result
data class Error(val message: String) : Result

// Kotlin 1.4+ context receivers (Kotlin 1.6.20+)
context(Logging)
fun logAndExecute(action: () -> Unit) {
    log("Starting")
    action()
    log("Done")
}

interface Logging {
    fun log(message: String)
}

// Usage with context declaration
context(Logging)
fun demonstrateValueClass() {
    val userId = UserId(12345L)
    val result: Result = Success("Done")

    when (result) {
        is Success -> println("Success: ${result.value}")
        is Error -> println("Error: ${result.message}")
    }
}
