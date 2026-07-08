package sample

// Kotlin expect/actual declarations (multiplatform)
expect class Platform() {
    val name: String
    fun getPlatformInfo(): String
}

// JVM implementation
actual class Platform actual constructor() {
    actual val name: String = "Java Virtual Machine"
    actual fun getPlatformInfo(): String = "Running on JVM"
}

// expect function
expect fun getOsName(): String

// actual function for JVM
actual fun getOsName(): String = System.getProperty("os.name")
