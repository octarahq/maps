package com.octara.maps.types

import com.google.gson.TypeAdapter
import com.google.gson.annotations.JsonAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter

@JsonAdapter(ParentType.Adapter::class)
enum class ParentType(val value: Int) {
    UNKNOWN(0),
    LIFT(1),
    METRO(2),
    STREET_BUS(3),
    STREET_TRAM(4),
    RAIL(5);

    companion object {
        fun fromValue(value: Int): ParentType =
            entries.firstOrNull { it.value == value } ?: UNKNOWN
    }

    class Adapter : TypeAdapter<ParentType>() {
        override fun write(out: JsonWriter, value: ParentType?) {
            out.value(value?.value ?: UNKNOWN.value)
        }

        override fun read(reader: JsonReader): ParentType {
            if (reader.peek() == JsonToken.NULL) {
                reader.nextNull()
                return UNKNOWN
            }
            return fromValue(reader.nextInt())
        }
    }
}

@JsonAdapter(RouteType.Adapter::class)
enum class RouteType(val value: Int) {
    TRAM(0),
    METRO(1),
    TRAIN(2),
    BUS(3),
    FERRY(4),
    CABLE_TRAM(5),
    AERIAL_LIFT(6),
    FUNICULAR(7);

    companion object {
        fun fromValue(value: Int): RouteType =
            entries.firstOrNull { it.value == value } ?: BUS
    }

    class Adapter : TypeAdapter<RouteType>() {
        override fun write(out: JsonWriter, value: RouteType?) {
            out.value(value?.value ?: BUS.value)
        }

        override fun read(reader: JsonReader): RouteType {
            if (reader.peek() == JsonToken.NULL) {
                reader.nextNull()
                return BUS
            }
            return fromValue(reader.nextInt())
        }
    }
}

data class ParentStopRoute(
    val id: String,
    val name: String,
    val type: RouteType,
    val color: String
)

data class ParentStop(
    val id: String,
    val name: String,
    val type: ParentType,
    val lat: Float,
    val lon: Float,
    val children_stops: List<String>,
    val routes: List<ParentStopRoute>
)