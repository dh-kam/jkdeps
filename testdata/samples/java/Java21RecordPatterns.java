package sample;

// Java 21 record patterns
public record Point(int x, int y) {}

public record Rectangle(Point upperLeft, Point lowerRight) {}

public class Java21Feature {
    // Record patterns in instanceof
    public void printUpperLeft(Object obj) {
        if (obj instanceof Rectangle(Point(int x, int y), _)) {
            System.out.println("Upper left: (" + x + ", " + y + ")");
        }
    }

    // Record patterns in switch
    public String describe(Object obj) {
        return switch (obj) {
            case Point(int x, int y) -> "Point at (" + x + ", " + y + ")";
            case null -> "null";
            default -> "Unknown";
        };
    }

    // Switch expressions with when guards
    public String classify(Object obj) {
        return switch (obj) {
            case String s when s.length() > 5 -> "Long string: " + s;
            case String s -> "Short string: " + s;
            default -> "Not a string";
        };
    }
}
