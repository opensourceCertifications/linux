# Use Debian or Ubuntu as the base image
FROM debian:latest

# Set environment variables to avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Update system and install necessary tools
RUN apt update && apt upgrade -y && \
    apt install -y sudo python3 python3-pip vim curl git unzip && \
    apt clean

ARG USERNAME="proper user name"

# Create a new user and grant sudo privileges
RUN useradd -m -s /bin/bash $USERNAME && echo "newuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Copy task and break lists into the container
# location is the same in both cases, have that as an env
ARG LOCATION=/home/$USERNAME
COPY todo.csv $LOCATION/todo.csv
COPY break.csv $LOCATION/break.csv

# Copy the setup script and make it executable
COPY setup.sh /home/newuser/setup.sh
RUN chmod +x /home/newuser/setup.sh

# Set user to "newuser" for when the container is run
USER $USERNAME

# Run the setup script when the container starts
# pass in location of break and todo csv
CMD ["/home/newuser/setup.sh $LOCATION"]

