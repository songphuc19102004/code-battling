import sys

# Print a message to stderr to show the program is running.
# This is useful for debugging and testing stderr redirection.
print("Processing input on stderr...", file=sys.stderr)

# Read a single line from stdin.
try:
    name = input()
except EOFError:
    name = "No one"


# Strip leading/trailing whitespace and print the greeting to stdout.
print(f"Hello, {name}!")
