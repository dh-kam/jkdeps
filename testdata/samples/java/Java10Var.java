package sample;

import java.util.List;

public class Java10Var {
    // Type inference (var)
    public void varExample() {
        var name = "Hello";
        var numbers = List.of(1, 2, 3);

        for (var n : numbers) {
            System.out.println(n);
        }
    }
}
