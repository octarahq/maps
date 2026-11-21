package com.octara.maps.ui.screens.navigation

import com.octara.maps.types.IntermediateStop
import com.octara.maps.types.LegType
import com.octara.maps.types.ParsedTrip
import com.octara.maps.types.RouteSummary
import com.octara.maps.types.RouteType
import com.octara.maps.types.TripLeg
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

class NavigationProgressTrackerTest {

    private fun iso(epochMs: Long): String =
        Instant.ofEpochMilli(epochMs).atOffset(ZoneOffset.ofHours(1)).format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)

    private fun sampleTrainTrip(t0: Long): ParsedTrip {

        val stop1 = IntermediateStop(
            sid = "s1",
            name = "Stop 1",
            arrivalTime = iso(t0 + 4 * 60_000L),
            departureTime = iso(t0 + 4 * 60_000L),
            lat = 48.8700,
            lon = 2.1200
        )
        val stop2 = IntermediateStop(
            sid = "s2",
            name = "Stop 2",
            arrivalTime = iso(t0 + 8 * 60_000L),
            departureTime = iso(t0 + 8 * 60_000L),
            lat = 48.8700,
            lon = 2.1400
        )
        val leg0 = TripLeg(
            type = LegType.TRANSIT,
            fromSID = "s0",
            toSID = "s3",
            fromStopName = "Depart",
            toStopName = "Terminus",
            fromLat = 48.8700,
            fromLon = 2.1000,
            toLat = 48.8700,
            toLon = 2.1600,
            departureTime = iso(t0),
            arrivalTime = iso(t0 + 12 * 60_000L),
            durationSec = 12 * 60,
            route = RouteSummary(rid = "L", shortName = "L", longName = "Ligne L", routeType = RouteType.TRAIN, color = "#6C72B5", textColor = "#FFFFFF"),
            headsign = "Terminus",
            intermediateStops = listOf(stop1, stop2)
        )
        return ParsedTrip(
            fromSID = "s0",
            toSID = "s3",
            departureTime = iso(t0),
            arrivalTime = iso(t0 + 12 * 60_000L),
            durationSec = 12 * 60,
            walkingDurationSec = 0,
            transfersCount = 0,
            legs = listOf(leg0)
        )
    }

    @Test
    fun gpsDrivesYellowDotBetweenStopsDespiteLateOrEarlySchedule() {
        val t0 = 1_700_000_000_000L
        val trip = sampleTrainTrip(t0)
        val tracker = NavigationSessionTracker()

        val waitingLate = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1002f,
            gpsFixMillis = t0 + 3 * 60_000L,
            nowMillis = t0 + 3 * 60_000L,
            tracker = tracker
        )
        assertEquals(0, waitingLate.activeStopPosition)
        assertTrue(waitingLate.isAtStop)
        assertTrue(waitingLate.nextActionTitle.startsWith("Prenez L"))

        val trackerMidEarly = NavigationSessionTracker()
        val between1And2WhenEarly = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1300f,
            gpsFixMillis = t0 + 1 * 60_000L,
            nowMillis = t0 + 1 * 60_000L,
            tracker = trackerMidEarly
        )
        assertEquals(2, between1And2WhenEarly.activeStopPosition)
        assertFalse(between1And2WhenEarly.isAtStop)
        assertTrue(between1And2WhenEarly.nextActionTitle.contains("Descendez"))

        val trackerMidLate = NavigationSessionTracker()
        val between1And2WhenLate = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1300f,
            gpsFixMillis = t0 + 11 * 60_000L,
            nowMillis = t0 + 11 * 60_000L,
            tracker = trackerMidLate
        )
        assertEquals(2, between1And2WhenLate.activeStopPosition)
        assertFalse(between1And2WhenLate.isAtStop)
    }

    @Test
    fun gpsDepartureCapturesLightDelayAndAppliesHalfCatchUpToNextStop() {
        val t0 = 1_700_000_000_000L
        val trip = sampleTrainTrip(t0)
        val tracker = NavigationSessionTracker()

        resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1003f,
            gpsFixMillis = t0 + 60_000L,
            nowMillis = t0 + 60_000L,
            tracker = tracker
        )

        val depTime = t0 + 2 * 60_000L
        val afterDep = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1040f,
            gpsFixMillis = depTime,
            nowMillis = depTime,
            tracker = tracker
        )

        assertEquals(1, afterDep.activeStopPosition)
        assertFalse(afterDep.isAtStop)

        val leg = afterDep.adjustedLegs[0]
        val s1ArrMs = parseIsoTimeMillis(leg.intermediateStops!![0].arrivalTime)!!
        val s2ArrMs = parseIsoTimeMillis(leg.intermediateStops!![1].arrivalTime)!!

        assertEquals(t0 + 5 * 60_000L, s1ArrMs)

        assertEquals(t0 + 8 * 60_000L + 30_000L, s2ArrMs)
    }

    @Test
    fun gpsDepartureCapturesLightAdvanceAndAppliesHalfSlowDownToNextStop() {
        val t0 = 1_700_000_000_000L
        val trip = sampleTrainTrip(t0)
        val tracker = NavigationSessionTracker()

        resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1002f,
            gpsFixMillis = t0 - 3 * 60_000L,
            nowMillis = t0 - 3 * 60_000L,
            tracker = tracker
        )

        val earlyDep = t0 - 2 * 60_000L
        val res = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1040f,
            gpsFixMillis = earlyDep,
            nowMillis = earlyDep,
            tracker = tracker
        )

        assertEquals(1, res.activeStopPosition)
        assertFalse(res.isAtStop)
        val s1ArrMs = parseIsoTimeMillis(res.adjustedLegs[0].intermediateStops!![0].arrivalTime)!!

        assertEquals(t0 + 3 * 60_000L, s1ArrMs)
    }

    @Test
    fun tunnelLossThenGpsReturnResyncsBackToTrueLegWithoutRestart() {
        val t0 = 1_700_000_000_000L
        val leg0 = TripLeg(
            type = LegType.TRANSIT,
            fromSID = "m0",
            toSID = "m1",
            fromStopName = "Villejuif",
            toStopName = "Saint-Lazare M14",
            fromLat = 48.7950,
            fromLon = 2.3480,
            toLat = 48.8757,
            toLon = 2.3243,
            departureTime = iso(t0),
            arrivalTime = iso(t0 + 25 * 60_000L),
            durationSec = 25 * 60,
            route = RouteSummary(rid = "14", shortName = "14", longName = "Métro 14", routeType = RouteType.METRO, color = "#62259D", textColor = "#FFFFFF")
        )
        val leg1 = TripLeg(
            type = LegType.TRANSFERT,
            fromSID = "m1",
            toSID = "l0",
            fromStopName = "Saint-Lazare",
            toStopName = "Gare Saint-Lazare",
            fromLat = 48.8757,
            fromLon = 2.3243,
            toLat = 48.8768,
            toLon = 2.3252,
            departureTime = iso(t0 + 25 * 60_000L),
            arrivalTime = iso(t0 + 30 * 60_000L),
            durationSec = 5 * 60
        )
        val leg2 = TripLeg(
            type = LegType.TRANSIT,
            fromSID = "l0",
            toSID = "l1",
            fromStopName = "Gare Saint-Lazare",
            toStopName = "Saint-Nom",
            fromLat = 48.8768,
            fromLon = 2.3252,
            toLat = 48.8679,
            toLon = 2.0501,
            departureTime = iso(t0 + 30 * 60_000L),
            arrivalTime = iso(t0 + 65 * 60_000L),
            durationSec = 35 * 60,
            route = RouteSummary(rid = "L", shortName = "L", longName = "Ligne L", routeType = RouteType.TRAIN, color = "#6C72B5", textColor = "#FFFFFF")
        )
        val trip = ParsedTrip(
            fromSID = "m0",
            toSID = "l1",
            departureTime = iso(t0),
            arrivalTime = iso(t0 + 65 * 60_000L),
            durationSec = 65 * 60,
            walkingDurationSec = 0,
            transfersCount = 1,
            legs = listOf(leg0, leg1, leg2)
        )
        val tracker = NavigationSessionTracker()

        val offlineRes = resolveTrackedNavigationProgress(
            trip = trip,
            location = null,
            gpsFixMillis = 0L,
            nowMillis = t0 + 35 * 60_000L,
            tracker = tracker
        )
        assertEquals(2, offlineRes.activeLegIndex)
        assertFalse(offlineRes.gpsSynced)

        val recoveredRes = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8350f to 2.3360f,
            gpsFixMillis = t0 + 36 * 60_000L,
            nowMillis = t0 + 36 * 60_000L,
            tracker = tracker
        )
        assertTrue(recoveredRes.gpsSynced)
        assertEquals(0, recoveredRes.activeLegIndex)
    }

    @Test
    fun platformHintsAndEarlyAdvanceFlagsUpdateSynchronously() {
        val t0 = 1_700_000_000_000L
        val base = sampleTrainTrip(t0)
        val walk = TripLeg(
            type = LegType.WALK,
            fromSID = "orig",
            toSID = "s0",
            fromStopName = "Origine",
            toStopName = "Depart",
            fromLat = 48.8700,
            fromLon = 2.0940,
            toLat = 48.8700,
            toLon = 2.1000,
            departureTime = iso(t0 - 5 * 60_000L),
            arrivalTime = iso(t0 - 1 * 60_000L),
            durationSec = 4 * 60,
            distanceKm = 0.45
        )
        val trip = base.copy(legs = listOf(walk, base.legs[0]))
        val tracker = NavigationSessionTracker()

        val duringWalk = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.0950f,
            gpsFixMillis = t0 - 3 * 60_000L,
            nowMillis = t0 - 3 * 60_000L,
            tracker = tracker,
            platformHints = mapOf(1 to LegPlatformHint(ref = "1", exitSide = "left"))
        )
        assertEquals(0, duringWalk.activeLegIndex)
        assertTrue(duringWalk.nextActionSubtitle?.contains("Arrivée Quai 1") == true)
        assertTrue(duringWalk.nextActionSubtitle?.contains("Sortie à gauche") == true)

        resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1001f,
            gpsFixMillis = t0 - 2 * 60_000L - 10_000L,
            nowMillis = t0 - 2 * 60_000L - 10_000L,
            tracker = tracker,
            platformHints = mapOf(1 to LegPlatformHint(ref = "1", exitSide = "left"))
        )
        val earlyLeft = resolveTrackedNavigationProgress(
            trip = trip,
            location = 48.8700f to 2.1040f,
            gpsFixMillis = t0 - 2 * 60_000L,
            nowMillis = t0 - 2 * 60_000L,
            tracker = tracker,
            platformHints = mapOf(1 to LegPlatformHint(ref = "1", exitSide = "left"))
        )
        assertEquals(1, earlyLeft.activeLegIndex)
        assertEquals(-60, earlyLeft.adjustedLegs[1].intermediateStops!![0].delaySec)
        assertTrue(earlyLeft.nextActionSubtitle?.contains("Quai 1") == true)
        assertTrue(earlyLeft.nextActionSubtitle?.contains("Sortie à gauche") == true)
    }
}
