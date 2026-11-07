package com.octara.maps.types

import com.google.gson.TypeAdapter
import com.google.gson.annotations.JsonAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter

@JsonAdapter(Accessibility.Adapter::class)
enum class Accessibility(val value: Int) {
    UNKNOWN(0),
    YES(1),
    NO(2);

    companion object {
        fun fromValue(value: Int): Accessibility =
            entries.firstOrNull { it.value == value } ?: UNKNOWN
    }

    class Adapter : TypeAdapter<Accessibility>() {
        override fun write(out: JsonWriter, value: Accessibility?) {
            out.value(value?.value ?: UNKNOWN.value)
        }

        override fun read(reader: JsonReader): Accessibility {
            if (reader.peek() == JsonToken.NULL) {
                reader.nextNull()
                return UNKNOWN
            }
            return fromValue(reader.nextInt())
        }
    }
}