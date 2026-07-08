package sample;

import java.util.function.Function;
import java.util.stream.Collectors;
import java.util.List;

public class Java8Lambda {
    // Lambda expression
    private Function<String, String> toUpper = s -> s.toUpperCase();

    // Method reference
    private List<String> transform(List<String> input) {
        return input.stream()
                .map(String::toUpperCase)
                .collect(Collectors.toList());
    }

    // Default method in interface (Java 8)
    interface MyInterface {
        default void doSomething() {
            System.out.println("Default method");
        }
    }
}
