package sample

// Kotlin 1.4+ fun interface (SAM interface)
fun interface ClickListener {
    fun onClick(x: Int, y: Int)
}

fun interface Transformer<T, R> {
    fun transform(value: T): R
}

// Usage
fun setupClickButton(listener: ClickListener) {
    listener.onClick(10, 20)
}

fun <T, R> mapValue(value: T, transformer: Transformer<T, R>): R {
    return transformer.transform(value)
}

// Extension function with functional interface
fun ClickListener.andThen(other: ClickListener): ClickListener {
    return ClickListener { x, y ->
        this.onClick(x, y)
        other.onClick(x, y)
    }
}
