// Java 9 module-info.java example
// Note: This would be in module-info.java, not a regular Java file

package sample.module;

// Java 9 private interface methods
public class Java9Feature {
    interface MyInterface {
        private void helper() {
            System.out.println("Helper");
        }

        default void doWork() {
            helper();
        }
    }

    // Diamond operator for anonymous inner class
    private MyInterface var = new MyInterface() {};
}
