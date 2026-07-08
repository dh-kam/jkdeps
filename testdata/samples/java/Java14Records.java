package sample;

// Java 14 records (preview in 14, final in 16)
public record Person(String name, int age) {
    // Compact constructor
    public Person {
        if (age < 0) {
            throw new IllegalArgumentException("Age cannot be negative");
        }
    }

    public String getDescription() {
        return name + " is " + age + " years old";
    }
}

// Java 14 instanceof pattern matching
public class Java14PatternMatching {
    public void process(Object obj) {
        if (obj instanceof String s) {
            System.out.println(s.toUpperCase());
        }
    }
}
