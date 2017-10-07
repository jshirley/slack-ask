# Configuration Settings for JIRA

The command `/ask link PROJKEY` will link the Ask App to a backend project in JIRA. Example usage is:

The idea is that all channels are tied to a specific project.

You can also set `/ask config type Task` to set the default issue type to Task. If the issue type isn't found in that
JIRA project, it will just use the default for the project.

# Why Mongo?

Because at my job we use Mongo for storing data like this!
