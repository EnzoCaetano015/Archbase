```java
package com.example.api.request;

import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.NonNull;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
public class UpdateItemRequest {

    @NonNull
    private String id;

    @NonNull
    private String name;

}
```
