+++
title = 'Guide: How to Mask Sensitive Information Using Fluent Bit'
date = 2024-01-21T10:39:08+05:30
draft = false
+++


Fluent Bit is a popular open-source log processor and forwarder that allows you to collect data from different sources, filter, and transform it before forwarding it to different destinations. In some cases, the data collected may contain sensitive information like passwords, credit card numbers, social security numbers, and other personal identifiable information (PII). To protect such information, you need to mask or obfuscate it before forwarding it to the destination. In this document, we will discuss how to mask sensitive information using Fluent Bit.

The goal of this guide is convert structured logs that contain PII information like (mobile numbers, identity information, name etc…)

`{"timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/user","message":"Sending SMS to mobileNumber=1234512345 registered on aadhaarNumber=1234512345"}` 
to a format where this information is masked.
`{"@timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/user","message":"Sending SMS to mobileNumber=******** registered on aadhaarNumber=********"}`

## Prerequisites & Constraints

This guides assumes the following

- Docker is installed on your machine
- You have knowledge of fluent-bit concepts like inputs, outputs, parsers, filters etc..
- We will be constraining our selves to not introduce any 3rd party service other than fluent-bit.

Let’s start with an initial configuration on your machine that reproduce the behavior where sensitive information is not masked yet.

Create an empty directory on your computer & save the below fluent-bit configuration in a file name `fluent-bit.conf`. 

```yaml
[INPUT]
    Name   dummy
    dummy  {"@timestamp":"2023-06-05T17:04:33.505+05:30","message":"Staring server on port 8080"}
    Tag    dummy.log

[INPUT]
    Name   dummy
    dummy  {"@timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/user","message":"Sending SMS to mobileNumber=1234512345, registered on aadhaarNumber=1234512345"}
    Tag    dummy.log

[INPUT]
    Name   dummy
    dummy  {"@timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/bank","message":"Successfully registered mobileNumber=1234512345, to panNumber=1234512345"}
    Tag    dummy.log

[OUTPUT]
    Name   stdout
    Match  *
```

Now run the below command from the same directory to start the fluent-bit process in a docker container.

```yaml
docker run \
  -v $(pwd)/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf \
  -ti cr.fluentbit.io/fluent/fluent-bit:2.0 \
  /fluent-bit/bin/fluent-bit \
  -c /fluent-bit/etc/fluent-bit.conf
```

After running the command, the expected output should look like this 👇

![initial-logs.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/1e224546-be83-45b9-bcb7-30b8e0740f81/initial-logs.png)

As you can see `mobileNumber` & `aadhaarNumber` are clearly visible in the application logs. Let’s start masking this information.

The problem of masking PII information can be logically solved by search & replace operation. Where we first search the required information from logs, once found we apply replace operation where the value of replace can be as simple as replacing with ******** or **hashing the value.**

In Fluent-Bit, the above mentioned operations can be performed at the Filter stage. Now, let’s select the right plugin for this stage.

## Selecting the Right Fluent-Bit Plugin

Out of the box fluent-bit provides `Nightfall` plugin, which interacts with a 3rd party component called [Nightfall](https://www.nightfall.ai/) to process the PII information. We won’t be using `Nightfall` plugin as we don’t want to introduce any 3rd party component in our system. 

Search & Replace can be performed by these 3 filter plugins:

- **Record Modifier**: This plugin gives us the ability to modify our structured logs by replacing entire key, values with something else. But does not allow replacing a small part of the value. For example, take the below structured log
`{"timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/user","message":"Sending SMS to mobileNumber=1234512345 registered on aadhaarNumber=1234512345"}`
in our case the `message` field of type string has `mobileNumber` in it. We just want to replace this value. But this filter can only replace the entire message content.
So, we won’t using this filter plugin
- **Modify**: This has the same drawback as `Record Modifier` plugin. So, we won’t using this filter plugin
- **Lua**: This filter allows you to modify the incoming records using custom [Lua](https://www.lua.org/) scripts. This helps us to extend Fluent Bit capabilities by writing custom filters using Lua programming language.
We can use this to write custom Lua script to perform search & replace operation for us.

## Writing the Lua Script

Create a file called `mask.lua` in the same directory where `fluent-bit.conf` exists. Copy the below content inside `mask.lua` file.

```lua
function mask_sensitive_info(tag, timestamp, record)
    message = record["message"]
    if message then
        -- Match "aadhaarNumber:xxxx," and replace with "aadhaarNumber:****,"
        local masked_message = string.gsub(message, 'aadhaarNumber=[^,]*', 'aadhaarNumber=****')

        -- Match "mobileNumber:xxxx," and replace with "mobileNumber:****,"
        masked_message = string.gsub(masked_message, 'mobileNumber=[^,]*', 'mobileNumber=****')

        record["message"] = masked_message
    end
    return 2, timestamp, record
end
```

Here's a breakdown of what this Lua function does:

1. `function mask_sensitive_info(tag, timestamp, record)`: This line defines a new Lua function named `mask_sensitive_info`. This function takes three parameters: `tag`, `timestamp`, and `record`.
2. `message = record["message"]`: This line retrieves the value of the "message" field from the `record` parameter and assigns it to a local variable named `message`.
3. `if message then`: This line starts an if statement that only executes the enclosed block of code if `message` is not `nil` or `false`.
4. `local masked_message = string.gsub(message, 'aadhaarNumber[^,]*', 'aadhaarNumber:****')`: This line creates a new local variable named `masked_message`. It assigns to this variable the result of calling the `string.gsub` function on `message`. The `string.gsub` function is used to replace all occurrences of the pattern 'aadhaarNumber[^,]*' (which matches the string "aadhaarNumber" followed by any sequence of characters that are not a comma) with the string 'aadhaarNumber:***'.
5. `masked_message = string.gsub(masked_message, 'mobileNumber[^,]*', 'mobileNumber:****')`: This line reassigns `masked_message` with the result of another call to `string.gsub`, this time replacing all occurrences of the pattern 'mobileNumber[^,]*' (which matches the string "mobileNumber" followed by any sequence of characters that are not a comma) with the string 'mobileNumber:***'.
6. `record["message"] = masked_message`: This line updates the "message" field of `record` with the value of `masked_message`, which at this point should have all sensitive data replaced with asterisks.
7. `end`: This line closes the if statement.
8. `return 2, timestamp, record`: This line specifies what the function should return when called. In this case, it returns three values: the number 2, the value of `timestamp`, and the updated `record`.
9. `end`: This line closes the function definition.

So, the overall purpose of this function is to replace any sensitive data (like Aadhaar numbers and mobile numbers) in the "message" field of a record with asterisks, for privacy reasons.

## Using the Lua Script In Lua Plugin

To enable the `lua` plugin, you need to add it to the Fluent Bit configuration file. Below is an example of how to configure the lua pugin to mask the PII field in the log data.

```yaml
[INPUT]
    Name   dummy
    dummy  {"@timestamp":"2023-06-05T17:04:33.505+05:30","message":"Staring server on port 8080"}
    Tag    dummy.log

[INPUT]
    Name   dummy
    dummy  {"@timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/user","message":"Sending SMS to mobileNumber=1234512345, registered on aadhaarNumber=1234512345"}
    Tag    dummy.log

[INPUT]
    Name   dummy
    dummy  {"@timestamp":"2023-06-05T17:04:33.505+05:30","requestURI":"/api/bank","message":"Successfully registered mobileNumber=1234512345, to panNumber=1234512345"}
    Tag    dummy.log

[FILTER]
    Name    lua
    Match   *
    call    mask_sensitive_info
    script  /fluent-bit/scripts/mask.lua

[OUTPUT]
    Name   stdout
    Match  *
```

In the above configuration, we have added the lua plugin in the filter state. We set the `Match` parameter to `*` to match all incoming log data. We then specify the `script` parameter to `/fluent-bit/scripts/mask.lua` which specifies the path to the lua script file . We also set the `call` parameter to `mask_sensitive_info` to specify the function name which has to be loaded from the lua script file.

### Testing

To test the above configuration, run below command

```
docker run \
  -v $(pwd)/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf \
  -v $(pwd)/mask.lua:/fluent-bit/scripts/mask.lua \
  -ti cr.fluentbit.io/fluent/fluent-bit:2.0 \
  /fluent-bit/bin/fluent-bit \
  -c /fluent-bit/etc/fluent-bit.conf
```

The expected output should contain masked valued like this 👇

![final-logs.png](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/77bcb6e3-e276-40c5-8c7a-2d03975d04d5/final-logs.png)

## Conclusion

Masking sensitive information is an important security measure that helps protect personal identifiable information from unauthorized access or disclosure. Fluent Bit provides a simple and effective way of masking sensitive information using the `lua` filter plugin. By configuring the plugin to mask specific fields or patterns in the log data, you can ensure that no sensitive information is leaked to external systems or logs.