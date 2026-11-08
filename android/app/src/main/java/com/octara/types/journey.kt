package com.octara.maps.types

import com.google.gson.TypeAdapter
import com.google.gson.annotations.JsonAdapter
import com.google.gson.annotations.SerializedName
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter

sealed interface Position {
    data object CurrentLocation : Position
    data class Coordinates(val lat: Double, val lon: Double) : Position
    data class ByStop(val stopId: String) : Position
}

enum class TimeType {
    DEPARTURE,
    ARRIVAL
}

data class JourneyQuery(
    val from: Position,
    val to: Position,
    val time: String? = null,
    val timeType: TimeType = TimeType.DEPARTURE,
    val allowedModes: List<RouteType>? = null,
    val withBike: Boolean = false,
    val wheelchair: Boolean = false,
    val currentGps: Pair<Double, Double>? = null
)

data class JourneyResponse(
    @SerializedName("journey") val journey: Map<String, ParsedTrip>? = null
) {
    val journeys: Map<String, ParsedTrip>
        get() = journey ?: emptyMap()
}

data class BikeParking(
    @SerializedName("id") val id: String? = null,
    @SerializedName("name") val name: String? = null,
    @SerializedName("lat") val lat: Double = 0.0,
    @SerializedName("lon") val lon: Double = 0.0,
    @SerializedName("capacity") val capacity: Int? = null
)

data class ParsedTrip(
    @SerializedName("from_sid") val fromSID: String,
    @SerializedName("to_sid") val toSID: String,
    @SerializedName("departure_time") val departureTime: String,
    @SerializedName("arrival_time") val arrivalTime: String,
    @SerializedName("duration_sec") val durationSec: Int,
    @SerializedName("walking_duration_sec") val walkingDurationSec: Int,
    @SerializedName("bike_duration_sec") val bikeDurationSec: Int = 0,
    @SerializedName("transfers_count") val transfersCount: Int,
    @SerializedName("is_wheelchair") val isWheelchair: Accessibility = Accessibility.UNKNOWN,
    @SerializedName("is_bikes_allowed") val isBikesAllowed: Accessibility = Accessibility.UNKNOWN,
    @SerializedName("bike_parking") val bikeParking: BikeParking? = null,
    @SerializedName("bike_parking_dist_m") val bikeParkingDistM: Double = 0.0,
    @SerializedName("legs") val legs: List<TripLeg> = emptyList()
)

data class TripLeg(
    @SerializedName("type") val type: LegType,
    @SerializedName("trip_index") val tripIndex: Long = 0,
    @SerializedName("from_sid") val fromSID: String,
    @SerializedName("to_sid") val toSID: String,
    @SerializedName("from_stop_name") val fromStopName: String,
    @SerializedName("to_stop_name") val toStopName: String,
    @SerializedName("from_lat") val fromLat: Double = 0.0,
    @SerializedName("from_lon") val fromLon: Double = 0.0,
    @SerializedName("to_lat") val toLat: Double = 0.0,
    @SerializedName("to_lon") val toLon: Double = 0.0,
    @SerializedName("departure_time") val departureTime: String,
    @SerializedName("arrival_time") val arrivalTime: String,
    @SerializedName("duration_sec") val durationSec: Int,
    @SerializedName("distance_km") val distanceKm: Double = 0.0,

    @SerializedName("route") val route: RouteSummary? = null,
    @SerializedName("headsign") val headsign: String? = null,
    @SerializedName("intermediate_stops") val intermediateStops: List<IntermediateStop>? = null,

    @SerializedName("position_in_train") val positionInTrain: TripPositionInTrain? = null,

    @SerializedName("wheelchair") val wheelchair: Accessibility = Accessibility.UNKNOWN,
    @SerializedName("bikes_allowed") val bikesAllowed: Accessibility = Accessibility.UNKNOWN,

    @SerializedName("coordinates") val coordinates: List<List<Double>>? = null
)

data class RouteSummary(
    @SerializedName("rid") val rid: String,
    @SerializedName("short_name") val shortName: String,
    @SerializedName("long_name") val longName: String,
    @SerializedName("route_type") val routeType: RouteType,
    @SerializedName("color") val color: String,
    @SerializedName("text_color") val textColor: String,
    @SerializedName("picto") val picto: String? = null,
    @SerializedName("vehicle_image") val vehicleImage: String? = null,
    @SerializedName("vehicle_carriage") val vehicleCarriage: String? = null
) {
    val vehiculeImageUrl: String?
        get() = if (!vehicleImage.isNullOrEmpty()) "https://omaps-calc.orionhost.app/images/${vehicleImage}" else null
}

data class IntermediateStop(
    @SerializedName("sid") val sid: String,
    @SerializedName("name") val name: String,
    @SerializedName("arrival_time") val arrivalTime: String,
    @SerializedName("departure_time") val departureTime: String,
    @SerializedName("is_blocked") val isBlocked: Boolean = false,
    @SerializedName("lat") val lat: Double = 0.0,
    @SerializedName("lon") val lon: Double = 0.0
)

data class TripPositionInTrain(
    @SerializedName("nb_wagons") val nbWagons: Int = 0,
    @SerializedName("wagon") val wagon: Int = 0,
    @SerializedName("pos_avg") val posAvg: TrainPosition = TrainPosition.UNKNOWN
)

@JsonAdapter(LegType.Adapter::class)
enum class LegType(val value: Int) {
    WALK(0),
    TRANSIT(1),
    BIKE(2);

    companion object {
        fun fromValue(value: Int): LegType = entries.firstOrNull { it.value == value } ?: WALK
    }

    class Adapter : TypeAdapter<LegType>() {
        override fun write(out: JsonWriter, value: LegType?) {
            out.value(value?.value ?: WALK.value)
        }

        override fun read(reader: JsonReader): LegType {
            if (reader.peek() == JsonToken.NULL) {
                reader.nextNull()
                return WALK
            }
            return fromValue(reader.nextInt())
        }
    }
}

@JsonAdapter(TrainPosition.Adapter::class)
enum class TrainPosition(val value: Int) {
    UNKNOWN(0),
    FRONT(1),
    MIDDLE(2),
    BACK(3);

    companion object {
        fun fromValue(value: Int): TrainPosition = entries.firstOrNull { it.value == value } ?: UNKNOWN
    }

    class Adapter : TypeAdapter<TrainPosition>() {
        override fun write(out: JsonWriter, value: TrainPosition?) {
            out.value(value?.value ?: UNKNOWN.value)
        }

        override fun read(reader: JsonReader): TrainPosition {
            if (reader.peek() == JsonToken.NULL) {
                reader.nextNull()
                return UNKNOWN
            }
            return fromValue(reader.nextInt())
        }
    }
}