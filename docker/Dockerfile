# Use Debian or Ubuntu as the base image
FROM debian:latest

# Set environment variables to avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Update system and install necessary tools
RUN apt update && apt upgrade -y && \
    apt install -y sudo python3 python3-pip vim curl git unzip && \
    apt clean

# Create a new user and grant sudo privileges
RUN useradd -m -s /bin/bash newuser && echo "newuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Copy task and break lists into the container
COPY todo.csv /home/newuser/todo.csv
COPY break.csv /home/newuser/break.csv

# Copy the setup script and make it executable
COPY setup.sh /home/newuser/setup.sh
RUN chmod +x /home/newuser/setup.sh

# Set user to "newuser" for when the container is run
USER newuser

# Run the setup script when the container starts
CMD ["/home/newuser/setup.sh"]

