import org.jetbrains.kotlin.cli.common.messages.MessageCollector;
import org.jetbrains.kotlin.cli.jvm.compiler.EnvironmentConfigFiles;
import org.jetbrains.kotlin.cli.jvm.compiler.KotlinCoreEnvironment;
import org.jetbrains.kotlin.com.intellij.openapi.Disposable;
import org.jetbrains.kotlin.com.intellij.openapi.util.Disposer;
import org.jetbrains.kotlin.com.intellij.psi.PsiErrorElement;
import org.jetbrains.kotlin.com.intellij.psi.util.PsiTreeUtil;
import org.jetbrains.kotlin.config.CommonConfigurationKeys;
import org.jetbrains.kotlin.config.CompilerConfiguration;
import org.jetbrains.kotlin.psi.KtClass;
import org.jetbrains.kotlin.psi.KtDeclaration;
import org.jetbrains.kotlin.psi.KtFile;
import org.jetbrains.kotlin.psi.KtImportDirective;
import org.jetbrains.kotlin.psi.KtNamedDeclaration;
import org.jetbrains.kotlin.psi.KtNamedFunction;
import org.jetbrains.kotlin.psi.KtObjectDeclaration;
import org.jetbrains.kotlin.psi.KtProperty;
import org.jetbrains.kotlin.psi.KtPsiFactory;
import org.jetbrains.kotlin.psi.KtTypeAlias;
import org.jetbrains.kotlin.resolve.ImportPath;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.Objects;
import java.util.stream.Stream;

public final class KotlinOfficialSnapshot {
    private KotlinOfficialSnapshot() {
    }

    private static final class DeclarationSnapshot {
        final String kind;
        final String name;

        DeclarationSnapshot(String kind, String name) {
            this.kind = kind;
            this.name = name;
        }
    }

    private static final class FileSnapshot {
        final String path;
        final String packageName;
        final List<String> imports;
        final List<DeclarationSnapshot> declarations;
        final int errorCount;
        final List<String> errors;

        FileSnapshot(
            String path,
            String packageName,
            List<String> imports,
            List<DeclarationSnapshot> declarations,
            int errorCount,
            List<String> errors
        ) {
            this.path = path;
            this.packageName = packageName;
            this.imports = imports;
            this.declarations = declarations;
            this.errorCount = errorCount;
            this.errors = errors;
        }
    }

    public static void main(String[] args) throws Exception {
        Args parsed = parseArgs(args);
        run(parsed);
    }

    private static void run(Args args) throws Exception {
        Path root = args.repoRoot.toAbsolutePath().normalize();
        List<Path> files = collectKotlinFiles(root, args.includeKts);

        Disposable disposable = Disposer.newDisposable("kotlin-official-snapshot");
        List<FileSnapshot> snapshots = new ArrayList<>(files.size());
        try {
            CompilerConfiguration config = new CompilerConfiguration();
            config.put(CommonConfigurationKeys.MODULE_NAME, "kotlin-official-snapshot");
            config.put(CommonConfigurationKeys.MESSAGE_COLLECTOR_KEY, MessageCollector.Companion.getNONE());

            KotlinCoreEnvironment env = KotlinCoreEnvironment.createForProduction(
                disposable,
                config,
                EnvironmentConfigFiles.JVM_CONFIG_FILES
            );

            KtPsiFactory factory = new KtPsiFactory(env.getProject(), false);
            for (Path path : files) {
                String source = Files.readString(path);
                KtFile ktFile = factory.createFile(path.toString(), source);
                snapshots.add(buildSnapshot(root, path, ktFile));
            }
        } finally {
            Disposer.dispose(disposable);
        }

        snapshots.sort(Comparator.comparing(file -> file.path));

        int filesWithErrors = 0;
        for (FileSnapshot snapshot : snapshots) {
            if (snapshot.errorCount > 0) {
                filesWithErrors++;
            }
        }

        String json = buildJson(root, snapshots, filesWithErrors);
        if (args.outPath == null) {
            System.out.println(json);
            return;
        }

        Path outPath = args.outPath.toAbsolutePath().normalize();
        Path parent = outPath.getParent();
        if (parent != null) {
            Files.createDirectories(parent);
        }
        Files.writeString(outPath, json + "\n", StandardCharsets.UTF_8);
        System.out.println("saved: " + outPath);
    }

    private static FileSnapshot buildSnapshot(Path root, Path path, KtFile ktFile) {
        String relPath = toUnixPath(root.relativize(path));

        String packageName = ktFile.getPackageName();
        if (packageName == null) {
            packageName = "";
        }

        List<String> imports = new ArrayList<>();
        for (KtImportDirective directive : ktFile.getImportDirectives()) {
            ImportPath importPath = directive.getImportPath();
            if (importPath == null) {
                continue;
            }
            imports.add(importPath.getPathStr());
        }
        Collections.sort(imports);

        List<DeclarationSnapshot> declarations = new ArrayList<>();
        for (KtDeclaration declaration : ktFile.getDeclarations()) {
            String kind = declarationKind(declaration);
            if (kind.isEmpty()) {
                continue;
            }
            String name = "";
            if (declaration instanceof KtNamedDeclaration namedDeclaration) {
                String declarationName = namedDeclaration.getName();
                if (declarationName != null) {
                    name = declarationName;
                }
            }
            if (name.isEmpty()) {
                continue;
            }
            declarations.add(new DeclarationSnapshot(kind, name));
        }
        declarations.sort(
            Comparator
                .comparing((DeclarationSnapshot declaration) -> declaration.kind)
                .thenComparing(declaration -> declaration.name)
        );

        Collection<PsiErrorElement> errorElements = PsiTreeUtil.findChildrenOfType(ktFile, PsiErrorElement.class);
        List<String> errors = new ArrayList<>(errorElements.size());
        for (PsiErrorElement error : errorElements) {
            String message = error.getErrorDescription();
            if (message == null) {
                message = "";
            }
            errors.add(message);
        }
        Collections.sort(errors);

        return new FileSnapshot(relPath, packageName, imports, declarations, errors.size(), errors);
    }

    private static String declarationKind(KtDeclaration declaration) {
        if (declaration instanceof KtClass klass) {
            if (klass.isInterface()) {
                return "interface";
            }
            return "class";
        }
        if (declaration instanceof KtObjectDeclaration) {
            return "object";
        }
        if (declaration instanceof KtNamedFunction) {
            return "function";
        }
        if (declaration instanceof KtProperty) {
            return "property";
        }
        if (declaration instanceof KtTypeAlias) {
            return "typealias";
        }
        return "";
    }

    private static List<Path> collectKotlinFiles(Path root, boolean includeKts) throws IOException {
        List<Path> files = new ArrayList<>();
        try (Stream<Path> stream = Files.walk(root)) {
            stream
                .filter(Files::isRegularFile)
                .filter(path -> {
                    String lower = path.getFileName().toString().toLowerCase(Locale.ROOT);
                    if (lower.endsWith(".kt")) {
                        return true;
                    }
                    return includeKts && lower.endsWith(".kts");
                })
                .forEach(files::add);
        }
        files.sort(Comparator.comparing(path -> toUnixPath(root.relativize(path))));
        return files;
    }

    private static String toUnixPath(Path path) {
        return path.toString().replace('\\', '/');
    }

    private static String buildJson(Path root, List<FileSnapshot> files, int filesWithErrors) {
        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append("\"generated_at_utc\":").append(quote(OffsetDateTime.now(ZoneOffset.UTC).toString())).append(',');
        sb.append("\"root\":").append(quote(root.toString())).append(',');
        sb.append("\"total_files\":").append(files.size()).append(',');
        sb.append("\"files_with_errors\":").append(filesWithErrors).append(',');
        sb.append("\"files\":[");
        for (int i = 0; i < files.size(); i++) {
            FileSnapshot file = files.get(i);
            if (i > 0) {
                sb.append(',');
            }
            sb.append('{');
            sb.append("\"path\":").append(quote(file.path)).append(',');
            sb.append("\"package_name\":").append(quote(file.packageName)).append(',');
            sb.append("\"imports\":");
            appendStringArray(sb, file.imports);
            sb.append(',');
            sb.append("\"declarations\":[");
            for (int j = 0; j < file.declarations.size(); j++) {
                DeclarationSnapshot declaration = file.declarations.get(j);
                if (j > 0) {
                    sb.append(',');
                }
                sb.append('{');
                sb.append("\"kind\":").append(quote(declaration.kind)).append(',');
                sb.append("\"name\":").append(quote(declaration.name));
                sb.append('}');
            }
            sb.append("]");
            sb.append(',');
            sb.append("\"error_count\":").append(file.errorCount).append(',');
            sb.append("\"errors\":");
            appendStringArray(sb, file.errors);
            sb.append('}');
        }
        sb.append(']');
        sb.append('}');
        return sb.toString();
    }

    private static void appendStringArray(StringBuilder sb, List<String> values) {
        sb.append('[');
        for (int i = 0; i < values.size(); i++) {
            if (i > 0) {
                sb.append(',');
            }
            sb.append(quote(values.get(i)));
        }
        sb.append(']');
    }

    private static String quote(String value) {
        if (value == null) {
            return "\"\"";
        }
        StringBuilder sb = new StringBuilder();
        sb.append('"');
        for (int i = 0; i < value.length(); i++) {
            char ch = value.charAt(i);
            switch (ch) {
                case '"' -> sb.append("\\\"");
                case '\\' -> sb.append("\\\\");
                case '\b' -> sb.append("\\b");
                case '\f' -> sb.append("\\f");
                case '\n' -> sb.append("\\n");
                case '\r' -> sb.append("\\r");
                case '\t' -> sb.append("\\t");
                default -> {
                    if (ch < 0x20) {
                        sb.append(String.format("\\u%04x", (int) ch));
                    } else {
                        sb.append(ch);
                    }
                }
            }
        }
        sb.append('"');
        return sb.toString();
    }

    private static final class Args {
        final Path repoRoot;
        final Path outPath;
        final boolean includeKts;

        Args(Path repoRoot, Path outPath, boolean includeKts) {
            this.repoRoot = repoRoot;
            this.outPath = outPath;
            this.includeKts = includeKts;
        }
    }

    private static Args parseArgs(String[] args) {
        Path repo = null;
        Path out = null;
        boolean includeKts = true;

        for (int i = 0; i < args.length; i++) {
            String arg = Objects.requireNonNull(args[i], "arg");
            switch (arg) {
                case "--repo" -> {
                    if (i + 1 >= args.length) {
                        usageAndExit("missing value for --repo");
                    }
                    repo = Path.of(args[++i]);
                }
                case "--out" -> {
                    if (i + 1 >= args.length) {
                        usageAndExit("missing value for --out");
                    }
                    out = Path.of(args[++i]);
                }
                case "--include-kts" -> {
                    if (i + 1 >= args.length) {
                        usageAndExit("missing value for --include-kts");
                    }
                    includeKts = parseBool(args[++i]);
                }
                case "-h", "--help" -> usageAndExit(null);
                default -> usageAndExit("unknown argument: " + arg);
            }
        }

        if (repo == null) {
            usageAndExit("--repo is required");
        }
        return new Args(repo, out, includeKts);
    }

    private static boolean parseBool(String value) {
        String normalized = value.trim().toLowerCase(Locale.ROOT);
        if (normalized.equals("1") || normalized.equals("true") || normalized.equals("yes")) {
            return true;
        }
        if (normalized.equals("0") || normalized.equals("false") || normalized.equals("no")) {
            return false;
        }
        usageAndExit("invalid boolean value: " + value);
        return false;
    }

    private static void usageAndExit(String error) {
        if (error != null && !error.isEmpty()) {
            System.err.println(error);
            System.err.println();
        }
        System.err.println("Usage:");
        System.err.println("  KotlinOfficialSnapshot --repo <path> [--out <json-path>] [--include-kts <true|false>]");
        System.exit(error == null ? 0 : 2);
    }
}
