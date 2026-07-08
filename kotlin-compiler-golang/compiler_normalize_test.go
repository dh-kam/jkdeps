package kotlincompilergolang

import (
	"strings"
	"testing"
)

func TestNormalizeKotlinSourceForANTLR_StripsPlatformModifiers(t *testing.T) {
	source := []byte(`internal actual fun foo(): Int = 1
public expect class Sample
`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, " actual ") {
		t.Fatalf("expected actual modifier to be removed: %q", normalized)
	}
	if strings.Contains(normalized, " expect ") {
		t.Fatalf("expected expect modifier to be removed: %q", normalized)
	}
	if !strings.Contains(normalized, "fun foo") {
		t.Fatalf("expected function declaration to remain: %q", normalized)
	}
	if !strings.Contains(normalized, "class Sample") {
		t.Fatalf("expected class declaration to remain: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_LeavesExpectFunctionCall(t *testing.T) {
	source := []byte(`fun test(): Int = expect(42)`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if !strings.Contains(normalized, "expect(42)") {
		t.Fatalf("expected function call to remain unchanged: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_StripsValueAndFunInterfaceAndContext(t *testing.T) {
	source := []byte(`value class Amount(val cents: Int)
fun interface Listener {
    fun onEvent()
}
context(UserScope)
fun resolve() = Unit
`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, "value class") {
		t.Fatalf("expected value modifier to be removed: %q", normalized)
	}
	if strings.Contains(normalized, "fun interface") {
		t.Fatalf("expected fun modifier before interface to be removed: %q", normalized)
	}
	if strings.Contains(normalized, "context(UserScope)") {
		t.Fatalf("expected simple context receiver line to be removed: %q", normalized)
	}
	if !strings.Contains(normalized, "class Amount") {
		t.Fatalf("expected class declaration to remain: %q", normalized)
	}
	if !strings.Contains(normalized, "interface Listener") {
		t.Fatalf("expected interface declaration to remain: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_StripsSealedLambdaLabelAndFunctionTypeCast(t *testing.T) {
	source := []byte(`sealed interface Marker

fun <T> sample(input: Any, f: (T) -> Unit) {
    input.fold(0) fold@{ _, _ -> 1 }
    val g = f as (Any?) -> Unit
    val h = f as suspend (Any?) -> Unit
    val r = input as Runnable?
}
`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, "sealed interface") {
		t.Fatalf("expected sealed modifier before interface to be removed: %q", normalized)
	}
	if strings.Contains(normalized, "fold@{") {
		t.Fatalf("expected lambda label before block to be removed: %q", normalized)
	}
	if strings.Contains(normalized, "as (Any?) -> Unit") {
		t.Fatalf("expected function type cast to be removed: %q", normalized)
	}
	if strings.Contains(normalized, "as suspend (Any?) -> Unit") {
		t.Fatalf("expected suspend function type cast to be removed: %q", normalized)
	}
	if strings.Contains(normalized, "as Runnable?") {
		t.Fatalf("expected nullable type cast to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "as Runnable") {
		t.Fatalf("expected normalized non-null cast to remain: %q", normalized)
	}
	if !strings.Contains(normalized, "interface Marker") {
		t.Fatalf("expected interface declaration to remain: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_NormalizesStringTemplateExpressions(t *testing.T) {
	source := []byte(`class Empty(private val isActive: Boolean) {
    override fun toString(): String = "Empty{${if (isActive) "Active" else "New" }}"
}`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, "${if") {
		t.Fatalf("expected complex template expression to be normalized: %q", normalized)
	}
	if strings.Contains(normalized, "${") {
		t.Fatalf("expected template expressions to be removed: %q", normalized)
	}
	if !strings.Contains(normalized, "\"Empty{0}\"") {
		t.Fatalf("expected normalized template placeholder: %q", normalized)
	}
}

func TestNormalizeObjectLiteralExpressions_ReplacesAnonymousObjectArguments(t *testing.T) {
	source := []byte(`fun sample(other: Any) {
    consume(object : Handler {
        override fun handle() = Unit
    })
    other.dumpTo(object : Collector {
        override fun collect() = Unit
    })
}
`)

	normalized := string(normalizeObjectLiteralExpressions(source))
	if strings.Contains(normalized, "object : Handler") || strings.Contains(normalized, "object : Collector") {
		t.Fatalf("expected anonymous object expressions to be elided: %q", normalized)
	}
	if !strings.Contains(normalized, "consume(null)") {
		t.Fatalf("expected anonymous object argument to become null: %q", normalized)
	}
	if !strings.Contains(normalized, "other.dumpTo(null)") {
		t.Fatalf("expected nested anonymous object argument to become null: %q", normalized)
	}
}

func TestNormalizeObjectLiteralExpressions_PreservesObjectDeclarations(t *testing.T) {
	source := []byte(`object NamedRegistry {
    fun install() = Unit
}

class Sample {
    companion object {
        fun create(): Sample = Sample()
    }
}
`)

	normalized := string(normalizeObjectLiteralExpressions(source))
	if normalized != string(source) {
		t.Fatalf("expected object declarations to remain unchanged: %q", normalized)
	}
}

func TestChoosePreferredParseOutcomePrefersFewerDiagnostics(t *testing.T) {
	primary := parseSourceOutcome{
		diagnostics: []Diagnostic{
			{Path: "a.kt", Message: "err1"},
			{Path: "a.kt", Message: "err2"},
		},
	}
	alternate := parseSourceOutcome{
		diagnostics: []Diagnostic{
			{Path: "a.kts", Message: "err1"},
		},
	}

	chosen := choosePreferredParseOutcome(primary, alternate)
	if len(chosen.diagnostics) != 1 {
		t.Fatalf("expected alternate outcome to be preferred, got %d diagnostics", len(chosen.diagnostics))
	}
}

func TestChoosePreferredParseOutcomeKeepsPrimaryOnTie(t *testing.T) {
	primary := parseSourceOutcome{
		diagnostics: []Diagnostic{
			{Path: "a.kt", Message: "err1"},
		},
	}
	alternate := parseSourceOutcome{
		diagnostics: []Diagnostic{
			{Path: "a.kts", Message: "err1"},
		},
	}

	chosen := choosePreferredParseOutcome(primary, alternate)
	if chosen.diagnostics[0].Path != "a.kt" {
		t.Fatalf("expected primary outcome to be kept on tie, got %q", chosen.diagnostics[0].Path)
	}
}

func TestNormalizeKnownTrailingLambdaCalls(t *testing.T) {
	source := []byte(`fun test(currentContext: Any, queue: Any, nextOrClosed: Any, x: Boolean) {
    currentContext.fold(0) { _, _ -> 1 }
    queue.loop { _ -> Unit }
    nextOrClosed.let { _ -> Unit }
    if (x) { println(x) }
}
`)

	normalized := string(normalizeKnownTrailingLambdaCalls(source))
	if !strings.Contains(normalized, "fold(0, { _, _ -> 1 })") {
		t.Fatalf("expected fold trailing lambda to become parenthesized arg: %q", normalized)
	}
	if !strings.Contains(normalized, "loop({ _ -> Unit })") {
		t.Fatalf("expected loop trailing lambda to become parenthesized arg: %q", normalized)
	}
	if !strings.Contains(normalized, "let({ _ -> Unit })") {
		t.Fatalf("expected let trailing lambda to become parenthesized arg: %q", normalized)
	}
	if !strings.Contains(normalized, "if (x) { println(x) }") {
		t.Fatalf("expected non-call block to remain unchanged: %q", normalized)
	}
}

func TestNormalizeKnownTrailingLambdaCalls_EventLoopStyle(t *testing.T) {
	source := []byte(`private fun dequeue(): Runnable? {
    _queue.loop { queue ->
      when (queue) {
        null -> return null
        is Queue<*> -> {
          val result = (queue as Queue<Runnable>).removeFirstOrNull()
          if (result !== Queue.REMOVE_FROZEN) return result as Runnable?
          _queue.compareAndSet(queue, queue.next())
        }
        else -> when {
          queue === CLOSED_EMPTY -> return null
          else -> if (_queue.compareAndSet(queue, null)) return queue as Runnable
        }
      }
    }
  }`)

	normalized := string(normalizeKnownTrailingLambdaCalls(source))
	if !strings.Contains(normalized, "_queue.loop({ queue ->") {
		t.Fatalf("expected loop trailing lambda to be transformed: %q", normalized)
	}
	if !strings.Contains(normalized, "}\n    })") {
		t.Fatalf("expected transformed loop call to be closed with ) after lambda: %q", normalized)
	}
}

func TestHasKnownBareTrailingLambdaCall(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "runTest call", text: "runTest { println(\"ok\") }", want: true},
		{name: "thread call", text: "thread { println(\"ok\") }", want: true},
		{name: "unknown callee", text: "custom { println(\"ok\") }", want: false},
		{name: "declaration keyword before callee", text: "fun runTest { println(\"ok\") }", want: false},
		{name: "control flow block", text: "if (ready) { println(\"ok\") }", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bracePos := strings.Index(tc.text, "{")
			if bracePos < 0 {
				t.Fatalf("test fixture missing brace: %q", tc.text)
			}
			if got := hasKnownBareTrailingLambdaCall(tc.text, bracePos); got != tc.want {
				t.Fatalf("hasKnownBareTrailingLambdaCall(%q, %d) = %v, want %v", tc.text, bracePos, got, tc.want)
			}
		})
	}
}

func TestNormalizeGenericTrailingLambdaCalls_BareAndParenCalls(t *testing.T) {
	source := []byte(`fun t() {
    runTest {
        launch {
            thread {
                println("ok")
            }
        }
    }
    dispatch_async(dispatch_get_global_queue(0, 0)) {
        println("x")
    }
    foo() {
        println("y")
    }
}
`)
	normalized := string(normalizeGenericTrailingLambdaCalls(source))
	if !strings.Contains(normalized, "runTest({") {
		t.Fatalf("expected bare runTest trailing lambda to be parenthesized: %q", normalized)
	}
	if !strings.Contains(normalized, "launch({") {
		t.Fatalf("expected bare launch trailing lambda to be parenthesized: %q", normalized)
	}
	if !strings.Contains(normalized, "thread({") {
		t.Fatalf("expected bare thread trailing lambda to be parenthesized: %q", normalized)
	}
	if !strings.Contains(normalized, "dispatch_async(dispatch_get_global_queue(0, 0), {") {
		t.Fatalf("expected paren-call trailing lambda to become explicit argument: %q", normalized)
	}
	if !strings.Contains(normalized, "foo({") {
		t.Fatalf("expected empty-arg call trailing lambda to become explicit argument: %q", normalized)
	}
}

func TestNormalizeGenericTrailingLambdaCalls_PreservesDeclarationsAndControlFlow(t *testing.T) {
	source := []byte(`fun f(x: Int) {
    if (x > 0) {
        println(x)
    }
}
class Derived : Base() {
    fun run() = Unit
}
class Box(v: Int) {
    fun run() {
        println(v)
    }
}
`)
	normalized := string(normalizeGenericTrailingLambdaCalls(source))
	if strings.Contains(normalized, "if (x > 0, {") || strings.Contains(normalized, "if (x > 0({") {
		t.Fatalf("did not expect control-flow blocks to be transformed: %q", normalized)
	}
	if strings.Contains(normalized, "class Box(v: Int, {") || strings.Contains(normalized, "class Box(v: Int({") {
		t.Fatalf("did not expect class declaration body to be transformed: %q", normalized)
	}
	if strings.Contains(normalized, "class Derived : Base({") || strings.Contains(normalized, "class Derived : Base, {") {
		t.Fatalf("did not expect supertype constructor in class header to be transformed: %q", normalized)
	}
	if strings.Contains(normalized, "fun f(x: Int, {") || strings.Contains(normalized, "fun f(x: Int({") {
		t.Fatalf("did not expect function declaration body to be transformed: %q", normalized)
	}
}

func TestShouldRewriteRunTestLambdaBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		opts genericTrailingOptions
		want bool
	}{
		{
			name: "simple body allowed",
			body: `println("ok")`,
			opts: genericTrailingOptions{rewriteRunTestSimpleOnly: true},
			want: true,
		},
		{
			name: "local function blocks rewrite",
			body: "fun helper() = Unit\nprintln(\"ok\")",
			opts: genericTrailingOptions{},
			want: false,
		},
		{
			name: "complex body blocked in simple-only mode",
			body: "\"hello\\n\" +\n    \"world\"",
			opts: genericTrailingOptions{rewriteRunTestSimpleOnly: true},
			want: false,
		},
		{
			name: "complex body allowed when not simple-only",
			body: "\"hello\\n\" +\n    \"world\"",
			opts: genericTrailingOptions{rewriteRunTestSimpleOnly: false},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRewriteRunTestLambdaBody(tc.body, tc.opts); got != tc.want {
				t.Fatalf("shouldRewriteRunTestLambdaBody(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestNormalizeDelegationConstructorsAndUnsignedLiterals(t *testing.T) {
	source := []byte(`class A: TestBase() {
    constructor(): this()
}
object B: CoroutineDispatcher()
val x = 0u
val y = 1U
`)
	normalized := string(normalizeDelegationConstructorsAndUnsignedLiterals(source))
	if strings.Contains(normalized, "class A: TestBase()") {
		t.Fatalf("expected empty delegation constructor call to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "class A: TestBase") {
		t.Fatalf("expected class delegation supertype to remain: %q", normalized)
	}
	if !strings.Contains(normalized, "constructor(): this()") {
		t.Fatalf("expected this() constructor delegation to be preserved: %q", normalized)
	}
	if strings.Contains(normalized, "object B: CoroutineDispatcher()") {
		t.Fatalf("expected object supertype constructor call to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "object B: CoroutineDispatcher") {
		t.Fatalf("expected normalized object supertype to remain: %q", normalized)
	}
	if strings.Contains(normalized, "0u") || strings.Contains(normalized, "1U") {
		t.Fatalf("expected unsigned literal suffixes to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "val x = 0") || !strings.Contains(normalized, "val y = 1") {
		t.Fatalf("expected unsigned literals to remain as numeric literals: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_StripsStarProjectionInExtensionReceiver(t *testing.T) {
	source := []byte(`internal fun SafeCollector<*>.checkContext(currentContext: CoroutineContext) {
    val collectJob = collectElement as Job?
    val safe = collectElement as? Job
}`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, "SafeCollector<*>.checkContext") {
		t.Fatalf("expected star projection in extension receiver to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "SafeCollector.checkContext") {
		t.Fatalf("expected extension receiver declaration to remain: %q", normalized)
	}
	if strings.Contains(normalized, "as Job?") {
		t.Fatalf("expected nullable cast to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "as? Job") {
		t.Fatalf("expected safe cast operator to remain unchanged: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_StripsGenericExtensionReceiver(t *testing.T) {
	source := []byte(`suspend fun <T : Any> Call<T>.await(): T {
  return suspendCancellableCoroutine { continuation ->
    continuation.invokeOnCancellation { cancel() }
  }
}`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, "Call<T>.await") {
		t.Fatalf("expected generic extension receiver to be normalized: %q", normalized)
	}
	if !strings.Contains(normalized, "Call.await") {
		t.Fatalf("expected extension receiver declaration to remain: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_ModernKotlinOperatorsAndSupertypes(t *testing.T) {
	source := []byte(`internal open class BufferedChannel<E>(
    private val capacity: Int
) : Channel<E> {
    fun trySend(element: E): ChannelResult<Unit> {
        return sendImpl(
            onSuspend = { segm, _ ->
                segm.onSlotCleaned()
                failure()
            }
        )
    }
}

internal open class SharedFlowImpl<T>(
    private val replay: Int
) : AbstractSharedFlow<SharedFlowSlot>(), MutableSharedFlow<T>, FusibleFlow<T> {
    fun correct(slot: Slot, newHead: Long) {
        if (slot.index in 0..<newHead) slot.index = newHead
    }
}`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	for _, banned := range []string{") : Channel<E>", "MutableSharedFlow<T>", "FusibleFlow<T>", "_ ->", "..<"} {
		if strings.Contains(normalized, banned) {
			t.Fatalf("expected %q to be normalized in: %q", banned, normalized)
		}
	}
	for _, want := range []string{"Channel {", "MutableSharedFlow", "FusibleFlow", "ignored ->", "0 until newHead"} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("expected %q in normalized source: %q", want, normalized)
		}
	}
}

func TestNormalizeKotlinSourceForANTLR_ModernFunctionTypes(t *testing.T) {
	source := []byte(`expect class Shared {
  actual companion object
}

internal val modules: MutableList<suspend Application.() -> Unit> = mutableListOf()

public inline fun <Error, A> either(
  block: context(Raise<Error>) () -> A
): Either<Error, A> = TODO()

fun cast(primaryFormat: Any, alternativeFormats: Any): Any {
  val a = primaryFormat as (AbstractDateTimeFormatBuilder<*, *>.() -> Unit)
  val b = alternativeFormats as Array<out AbstractDateTimeFormatBuilder<*, *>.() -> Unit>
  return a as T & Any
}
`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	for _, banned := range []string{
		"actual companion",
		"suspend Application.() ->",
		"context(Raise<Error>)",
		"AbstractDateTimeFormatBuilder<*, *>.() -> Unit",
		"& Any",
	} {
		if strings.Contains(normalized, banned) {
			t.Fatalf("expected %q to be normalized in: %q", banned, normalized)
		}
	}
	for _, want := range []string{
		"companion object",
		"MutableList<() -> Unit>",
		"block: () -> A",
		"as Array",
	} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("expected %q in normalized source: %q", want, normalized)
		}
	}
}

func TestIsLikelyANTLRCompatibilityGap(t *testing.T) {
	diagnostics := []Diagnostic{{Message: "antlr parse error", Severity: SeverityError}}
	modern := []byte(`fun sample(block: context(Raise<String>) () -> Unit) = block`)
	if !isLikelyANTLRCompatibilityGap(modern, diagnostics) {
		t.Fatalf("expected modern Kotlin syntax to be classified as ANTLR compatibility gap")
	}

	for _, source := range []string{
		"fun setup() {\n  value =\n    DetailRepositoryImpl(client)\n}",
		"class Interpolator :\n  BaseInterpolator<\n    Entry,\n    Model,\n  >()",
		"NavHost(navController) {\n  composable(route = route) { Screen() }\n}",
		"fun load() { try { call() } catch (t: Throwable) { throw t } }",
	} {
		if !isLikelyANTLRCompatibilityGap([]byte(source), diagnostics) {
			t.Fatalf("expected compatibility classification for source: %s", source)
		}
	}

	broken := []byte("fun broken(")
	if isLikelyANTLRCompatibilityGap(broken, diagnostics) {
		t.Fatalf("did not expect clearly incomplete Kotlin to be classified as compatibility gap")
	}
}

func TestIsKotlinScriptPath(t *testing.T) {
	if !isKotlinScriptPath("/tmp/build.gradle.kts") {
		t.Fatalf("expected .kts path to be recognized as script")
	}
	if !isKotlinScriptPath("/tmp/BUILD.GRADLE.KTS") {
		t.Fatalf("expected uppercase extension to be recognized as script")
	}
	if isKotlinScriptPath("/tmp/App.kt") {
		t.Fatalf("did not expect .kt path to be recognized as script")
	}
}

func TestStripAnnotationArguments(t *testing.T) {
	source := "@Suppress(\"UNCHECKED_CAST\", \"INVISIBLE_REFERENCE\")\n" +
		"@file:OptIn(ExperimentalCoroutinesApi::class)\n" +
		"class Sample\n"
	normalized := stripAnnotationArguments(source)
	if strings.Contains(normalized, "\"UNCHECKED_CAST\"") {
		t.Fatalf("expected annotation arguments to be stripped: %q", normalized)
	}
	if strings.Contains(normalized, "ExperimentalCoroutinesApi::class") {
		t.Fatalf("expected file annotation arguments to be stripped: %q", normalized)
	}
	if !strings.Contains(normalized, "@Suppress()") {
		t.Fatalf("expected annotation marker to remain: %q", normalized)
	}
	if !strings.Contains(normalized, "@file:OptIn()") {
		t.Fatalf("expected file annotation marker to remain: %q", normalized)
	}
}

func TestStripAnnotationArguments_PreservesLabelSyntax(t *testing.T) {
	source := "retry@while (true) {\n" +
		"  continue@retry\n" +
		"}\n" +
		"@Suppress(\"UNCHECKED_CAST\")\n" +
		"suspendCancellableCoroutine<Unit> sc@{ return@sc }\n"
	normalized := stripAnnotationArguments(source)
	if !strings.Contains(normalized, "retry@while (true)") {
		t.Fatalf("expected loop label syntax to be preserved: %q", normalized)
	}
	if !strings.Contains(normalized, "continue@retry") {
		t.Fatalf("expected continue label syntax to be preserved: %q", normalized)
	}
	if !strings.Contains(normalized, "return@sc") {
		t.Fatalf("expected return label syntax to be preserved: %q", normalized)
	}
	if !strings.Contains(normalized, "@Suppress()") {
		t.Fatalf("expected real annotation arguments to still be stripped: %q", normalized)
	}
}

func TestNormalizeKotlinSourceForANTLR_NormalizesAnonymousFunctionAssignments(t *testing.T) {
	source := []byte(`private val countAll =
    fun (countOrElement: Any?, element: CoroutineContext.Element): Any? {
        return countOrElement
    }
private val findOne =
    fun (found: ThreadContextElement<*>?, element: CoroutineContext.Element): ThreadContextElement<*>? {
        return element as? ThreadContextElement<*>
    }
`)
	normalized := string(normalizeKotlinSourceForANTLR(source))
	if strings.Contains(normalized, "= \n    fun (") || strings.Contains(normalized, "= fun (") {
		t.Fatalf("expected anonymous function assignments to be normalized to lambdas: %q", normalized)
	}
	if !strings.Contains(normalized, "= { countOrElement: Any?, element: CoroutineContext.Element ->") {
		t.Fatalf("expected first anonymous function assignment to become lambda: %q", normalized)
	}
	if !strings.Contains(normalized, "= { found: ThreadContextElement<*>?, element: CoroutineContext.Element ->") {
		t.Fatalf("expected second anonymous function assignment to become lambda: %q", normalized)
	}
}

func TestNormalizeParenthesizedLambdaBodies_RewritesMultilineBody(t *testing.T) {
	source := []byte(`fun f(window: dynamic) {
    window.addEventListener("message", { event: dynamic ->
        if (event.source == window) {
            event.stopPropagation()
            process()
        }
    }, true)
}
`)
	normalized := string(normalizeParenthesizedLambdaBodies(source))
	if !strings.Contains(normalized, "event.stopPropagation();") {
		t.Fatalf("expected multiline lambda body to be flattened with semicolons: %q", normalized)
	}
	if !strings.Contains(normalized, "process();") {
		t.Fatalf("expected subsequent lambda statement separator to be inserted: %q", normalized)
	}
}

func TestNormalizeParenthesizedLambdaBodies_PreservesLineCommentBoundaries(t *testing.T) {
	source := []byte(`fun f(run: (Int) -> Unit) {
    run({
        // keep this comment as its own line
        consume(1)
        consume(2)
    })
}
`)
	normalized := string(normalizeParenthesizedLambdaBodies(source))
	if !strings.Contains(normalized, "// keep this comment as its own line\n") {
		t.Fatalf("expected line comment newline to be preserved: %q", normalized)
	}
	if !strings.Contains(normalized, "consume(1);") || !strings.Contains(normalized, "consume(2);") {
		t.Fatalf("expected statements after comment to remain separated: %q", normalized)
	}
}

func TestNormalizeParenthesizedLambdaBodies_LeavesRegularMultilineParameters(t *testing.T) {
	source := []byte(`public fun <T> CoroutineScope.promise(
    context: CoroutineContext = EmptyCoroutineContext,
    start: CoroutineStart = CoroutineStart.DEFAULT,
    block: suspend CoroutineScope.() -> T
): Promise<T> = async(context, start, block).asPromise()
`)
	normalized := string(normalizeParenthesizedLambdaBodies(source))
	if normalized != string(source) {
		t.Fatalf("expected non-lambda multiline parameter list to remain unchanged:\nwant:\n%s\ngot:\n%s", string(source), normalized)
	}
}

func TestHasLikelyAnnotationArgumentInSource_DetectsAnnotationWithArguments(t *testing.T) {
	// Source with annotations that have arguments
	source := "@" + "Entity(table = \"users\")\nclass User"
	if !hasLikelyAnnotationArgumentInSource([]byte(source)) {
		t.Fatal("expected to detect annotation with arguments")
	}
}

func TestHasLikelyAnnotationArgumentInSource_IgnoresAnnotationWithoutArguments(t *testing.T) {
	// Source with annotations without arguments
	source := "@" + "Entity\n@" + "Serializable\nclass User"
	if hasLikelyAnnotationArgumentInSource([]byte(source)) {
		t.Fatal("expected no annotation arguments to be detected")
	}
}

func TestHasLikelyAnnotationArgumentInSource_DetectsNestedAnnotationCalls(t *testing.T) {
	// Nested annotation with arguments
	source := "@" + "Annotation(Other(\"value\"))\nclass Test"
	if !hasLikelyAnnotationArgumentInSource([]byte(source)) {
		t.Fatal("expected to detect nested annotation with arguments")
	}
}
