package com.octara.maps.types

import com.google.gson.TypeAdapter
import com.google.gson.annotations.JsonAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter

@JsonAdapter(StopType.Adapter::class)
enum class StopType(val value: Int) {
    UNKNOWN(0),
    LIFT(1),
    METRO(2),
    STREET_BUS(3),
    STREET_TRAM(4),
    RAIL(5);

    companion object {
        fun fromValue(value: Int): StopType =
            entries.firstOrNull { it.value == value } ?: UNKNOWN
    }

    class Adapter : TypeAdapter<StopType>() {
        override fun write(out: JsonWriter, value: StopType?) {
            out.value(value?.value ?: UNKNOWN.value)
        }

        override fun read(reader: JsonReader): StopType {
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
    val type: StopType,
    val lat: Float,
    val lon: Float,
    val children_stops: List<String>,
    val routes: List<ParentStopRoute>
)

data class Stop(
    val id: String,
    val name: String,
    val lat: Float,
    val lon: Float,
    val zone: String,
    val parent_station: String,
    val wheelchair: Accessibility,
)

data class Departure(
    val route_id: String,
    val route_name: String,
    val color: String,
    val type: RouteType,
    val headsign: String,
    val dep_sec: Int,
    val time: String,
    val time_string: String,
    val wait_minutes: Float,
)